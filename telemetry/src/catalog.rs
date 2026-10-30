use std::collections::HashMap;
use std::sync::OnceLock;

use prost::Message;
use prost_reflect::{
    DescriptorPool, DynamicMessage, Kind, MapKey, MessageDescriptor, ReflectMessage, Value,
};

use crate::config::SCHEMA_VERSION;

const SERVER_EVENTS: &[&str] = &[
    "auction_request_received",
    "auction_completed",
    "dsp_request_sent",
    "dsp_response_received",
    "dsp_response_rejected",
];

#[derive(Clone)]
struct EventType {
    descriptor: MessageDescriptor,
    full_name: String,
    indexes: Vec<u64>,
}

#[derive(Clone)]
pub struct Catalog {
    by_name: HashMap<String, EventType>,
}

#[derive(Debug)]
pub struct EncodedEvent {
    pub event_name: String,
    pub message_type: String,
    pub indexes: Vec<u64>,
    pub payload: Vec<u8>,
}

impl Catalog {
    pub fn load() -> Self {
        static CATALOG: OnceLock<Catalog> = OnceLock::new();
        CATALOG.get_or_init(load_catalog).clone()
    }

    pub fn contains(&self, event_name: &str) -> bool {
        self.by_name.contains_key(event_name)
    }

    #[cfg(test)]
    pub fn len(&self) -> usize {
        self.by_name.len()
    }

    pub fn subjects(&self, topic: &str) -> Vec<String> {
        self.by_name
            .values()
            .map(|event| format!("{topic}-{}", event.full_name))
            .collect()
    }

    /// Decodes one protojson client event and stamps server envelope fields.
    /// Unknown names and events that do not match their message return None.
    pub fn encode_event(
        &self,
        raw: &serde_json::Value,
        app_key: &str,
        bundle: &str,
        session_id: &str,
    ) -> Option<EncodedEvent> {
        let event_name = raw
            .get("envelope")
            .and_then(|envelope| envelope.get("event_name"))
            .and_then(|name| name.as_str())?;
        let event = self.by_name.get(event_name)?;
        if !identity_present(raw) {
            return None;
        }
        let mut msg = DynamicMessage::new(event.descriptor.clone());
        fill_message(&mut msg, raw).ok()?;
        stamp(&mut msg, app_key, bundle, session_id, raw, event_name).ok()?;
        Some(EncodedEvent {
            event_name: event_name.to_owned(),
            message_type: event.full_name.clone(),
            indexes: event.indexes.clone(),
            payload: msg.encode_to_vec(),
        })
    }

    #[cfg(test)]
    pub fn decode(&self, message_type: &str, payload: &[u8]) -> Option<DynamicMessage> {
        let event = self
            .by_name
            .values()
            .find(|event| event.full_name == message_type)?;
        DynamicMessage::decode(event.descriptor.clone(), payload).ok()
    }
}

fn load_catalog() -> Catalog {
    let bytes = include_bytes!(concat!(env!("OUT_DIR"), "/events.binpb"));
    let pool = DescriptorPool::decode(bytes.as_slice()).expect("events descriptor");
    let file = pool
        .get_file_by_name("org/bidon/telemetry/v1/events.proto")
        .expect("events.proto");
    let extension = pool
        .get_extension_by_name("org.bidon.telemetry.v1.event_name")
        .expect("event_name extension");
    let mut by_name = HashMap::new();
    for descriptor in file.messages() {
        let options = descriptor.options();
        let extension_value = options.get_extension(&extension);
        let Value::String(name) = extension_value.as_ref() else {
            continue;
        };
        if name.is_empty() || SERVER_EVENTS.contains(&name.as_str()) {
            continue;
        }
        let name = name.clone();
        by_name.insert(
            name,
            EventType {
                full_name: descriptor.full_name().to_owned(),
                indexes: message_indexes(&descriptor),
                descriptor,
            },
        );
    }
    Catalog { by_name }
}

fn message_indexes(descriptor: &MessageDescriptor) -> Vec<u64> {
    let mut chain = Vec::new();
    let mut current = descriptor.clone();
    loop {
        let index = match current.parent_message() {
            Some(parent) => position(&current, parent.child_messages()),
            None => position(&current, current.parent_file().messages()),
        };
        chain.push(index);
        match current.parent_message() {
            Some(parent) => current = parent,
            None => break,
        }
    }
    chain.reverse();
    chain
}

fn position(
    descriptor: &MessageDescriptor,
    mut siblings: impl Iterator<Item = MessageDescriptor>,
) -> u64 {
    siblings
        .position(|message| message.full_name() == descriptor.full_name())
        .expect("message index") as u64
}

fn identity_present(raw: &serde_json::Value) -> bool {
    let Some(envelope) = raw.get("envelope").and_then(|value| value.as_object()) else {
        return false;
    };
    let id = envelope
        .get("event_id")
        .and_then(|value| value.as_str())
        .unwrap_or("");
    envelope
        .get("event_ts")
        .and_then(|value| value.as_i64())
        .is_some()
        && !id.is_empty()
}

fn stamp(
    msg: &mut DynamicMessage,
    app_key: &str,
    bundle: &str,
    session_id: &str,
    raw: &serde_json::Value,
    event_name: &str,
) -> Result<(), ()> {
    let field = msg.descriptor().get_field_by_name("envelope").ok_or(())?;
    let mut envelope = match msg.get_field(&field).into_owned() {
        Value::Message(existing) => existing,
        _ => {
            let Kind::Message(descriptor) = field.kind() else {
                return Err(());
            };
            DynamicMessage::new(descriptor)
        }
    };
    set_i64(&mut envelope, "app_id", 0)?;
    set_string(&mut envelope, "app_key", app_key)?;
    set_string(&mut envelope, "bundle", bundle)?;
    set_string(&mut envelope, "schema_version", SCHEMA_VERSION)?;
    set_string(&mut envelope, "event_name", event_name)?;
    set_string(&mut envelope, "country", "")?;
    set_string(&mut envelope, "trace_id", "")?;
    let session_override = raw
        .get("envelope")
        .and_then(|envelope| envelope.get("session_id"))
        .and_then(|value| value.as_str())
        .is_some_and(|value| !value.is_empty());
    if !session_override {
        set_string(&mut envelope, "session_id", session_id)?;
    }
    let sampling_present = raw
        .get("envelope")
        .and_then(|envelope| envelope.get("sampling_rate"))
        .is_some();
    if !sampling_present {
        set_f64(&mut envelope, "sampling_rate", 1.0)?;
    }
    msg.set_field(&field, Value::Message(envelope));
    Ok(())
}

fn set_string(msg: &mut DynamicMessage, name: &str, value: &str) -> Result<(), ()> {
    let field = msg.descriptor().get_field_by_name(name).ok_or(())?;
    msg.set_field(&field, Value::String(value.to_owned()));
    Ok(())
}

fn set_i64(msg: &mut DynamicMessage, name: &str, value: i64) -> Result<(), ()> {
    let field = msg.descriptor().get_field_by_name(name).ok_or(())?;
    msg.set_field(&field, Value::I64(value));
    Ok(())
}

fn set_f64(msg: &mut DynamicMessage, name: &str, value: f64) -> Result<(), ()> {
    let field = msg.descriptor().get_field_by_name(name).ok_or(())?;
    msg.set_field(&field, Value::F64(value));
    Ok(())
}

fn fill_message(msg: &mut DynamicMessage, raw: &serde_json::Value) -> Result<(), ()> {
    let object = raw.as_object().ok_or(())?;
    for (key, value) in object {
        if value.is_null() {
            continue;
        }
        let Some(field) = msg.descriptor().get_field_by_name(key) else {
            continue;
        };
        if field.is_map() {
            msg.set_field(&field, Value::Map(json_string_map(value)?));
            continue;
        }
        if field.is_list() {
            msg.set_field(&field, Value::List(json_list(&field, value)?));
            continue;
        }
        if let Kind::Message(descriptor) = field.kind() {
            let mut nested = DynamicMessage::new(descriptor);
            fill_message(&mut nested, value)?;
            msg.set_field(&field, Value::Message(nested));
            continue;
        }
        msg.set_field(&field, json_scalar(&field.kind(), value)?);
    }
    Ok(())
}

fn json_list(
    field: &prost_reflect::FieldDescriptor,
    value: &serde_json::Value,
) -> Result<Vec<Value>, ()> {
    let items = value.as_array().ok_or(())?;
    let mut out = Vec::with_capacity(items.len());
    for item in items {
        out.push(json_value(&field.kind(), item)?);
    }
    Ok(out)
}

fn json_value(kind: &Kind, value: &serde_json::Value) -> Result<Value, ()> {
    if let Kind::Message(descriptor) = kind {
        let mut nested = DynamicMessage::new(descriptor.clone());
        fill_message(&mut nested, value)?;
        return Ok(Value::Message(nested));
    }
    json_scalar(kind, value)
}

fn json_string_map(value: &serde_json::Value) -> Result<HashMap<MapKey, Value>, ()> {
    let object = value.as_object().ok_or(())?;
    let mut map = HashMap::with_capacity(object.len());
    for (key, item) in object {
        let text = item.as_str().ok_or(())?;
        map.insert(MapKey::String(key.clone()), Value::String(text.to_owned()));
    }
    Ok(map)
}

fn json_scalar(kind: &Kind, value: &serde_json::Value) -> Result<Value, ()> {
    match kind {
        Kind::String => Ok(Value::String(value.as_str().ok_or(())?.to_owned())),
        Kind::Bool => Ok(Value::Bool(value.as_bool().ok_or(())?)),
        Kind::Int32 | Kind::Sint32 | Kind::Sfixed32 => Ok(Value::I32(
            json_i64(value).ok_or(())?.try_into().map_err(|_| ())?,
        )),
        Kind::Int64 | Kind::Sint64 | Kind::Sfixed64 => Ok(Value::I64(json_i64(value).ok_or(())?)),
        Kind::Double | Kind::Float => Ok(Value::F64(json_f64(value).ok_or(())?)),
        _ => Err(()),
    }
}

fn json_i64(value: &serde_json::Value) -> Option<i64> {
    value.as_i64().or_else(|| {
        let number = value.as_f64()?;
        if number.fract() == 0.0 {
            Some(number as i64)
        } else {
            None
        }
    })
}

fn json_f64(value: &serde_json::Value) -> Option<f64> {
    value
        .as_f64()
        .or_else(|| value.as_i64().map(|number| number as f64))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn client_catalog_excludes_server_events() {
        let catalog = Catalog::load();
        assert_eq!(catalog.len(), 43);
        assert!(catalog.contains("ad_request_started"));
        assert!(catalog.contains("sdk_crashed"));
        assert!(!catalog.contains("render_process_gone"));
        assert!(catalog.contains("signal_timeout_suspected"));
        assert!(!catalog.contains("dsp_request_sent"));
        assert!(!catalog.contains("auction_request_received"));
    }

    #[test]
    fn stamps_server_fields_and_drops_client_identity() {
        let catalog = Catalog::load();
        let raw = serde_json::json!({
            "envelope": {
                "event_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
                "event_name": "ad_request_started",
                "event_ts": 1710000000123i64,
                "auction_id": "auc_01J8Z3K4N5",
                "ad_type": "banner",
                "ad_format": "BANNER",
                "app_id": 1,
                "country": "US",
                "schema_version": "client",
                "trace_id": "client-trace"
            },
            "price_floor": 0.05,
            "ignored": true
        });
        let encoded = catalog
            .encode_event(
                &raw,
                "chess_2000",
                "com.demo.chessmaster",
                "550e8400-e29b-41d4-a716-446655440000",
            )
            .expect("encode");
        assert_eq!(encoded.event_name, "ad_request_started");
        assert_eq!(
            encoded.message_type,
            "org.bidon.telemetry.v1.AdRequestStarted"
        );
        assert_eq!(encoded.indexes, vec![14]);
        let decoded = catalog
            .decode(&encoded.message_type, &encoded.payload)
            .expect("decode");
        let envelope = match decoded.get_field_by_name("envelope").unwrap().into_owned() {
            Value::Message(envelope) => envelope,
            other => panic!("envelope: {other:?}"),
        };
        assert_eq!(field_i64(&envelope, "app_id"), 0);
        assert_eq!(field_string(&envelope, "app_key"), "chess_2000");
        assert_eq!(field_string(&envelope, "bundle"), "com.demo.chessmaster");
        assert_eq!(
            field_string(&envelope, "session_id"),
            "550e8400-e29b-41d4-a716-446655440000"
        );
        assert_eq!(field_string(&envelope, "schema_version"), "1");
        assert_eq!(field_string(&envelope, "country"), "");
        assert_eq!(field_string(&envelope, "trace_id"), "");
        assert_eq!(field_string(&envelope, "auction_id"), "auc_01J8Z3K4N5");
        match decoded.get_field_by_name("price_floor").unwrap().as_ref() {
            Value::F64(price) => assert!((*price - 0.05).abs() < f64::EPSILON),
            other => panic!("price_floor: {other:?}"),
        }
        assert!(decoded.get_field_by_name("ignored").is_none());
    }

    #[test]
    fn keeps_declared_fields_and_drops_ones_missing_from_the_proto() {
        let catalog = Catalog::load();
        let raw = serde_json::json!({
            "envelope": {
                "event_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
                "event_name": "ad_load_failed",
                "event_ts": 1710000000123i64
            },
            "error_code": "no_fill",
            "external_error_code": "3",
            "external_error_source": "admob",
            "exception_class": "LoadException",
            "exception_cause_class": "IOException",
            "not_in_proto": "dropped"
        });
        let encoded = catalog
            .encode_event(&raw, "chess_2000", "com.demo.chessmaster", "session")
            .expect("encode");
        let decoded = catalog
            .decode(&encoded.message_type, &encoded.payload)
            .expect("decode");
        assert_eq!(field_string(&decoded, "external_error_code"), "3");
        assert_eq!(field_string(&decoded, "external_error_source"), "admob");
        assert_eq!(field_string(&decoded, "exception_class"), "LoadException");
        assert_eq!(field_string(&decoded, "exception_cause_class"), "IOException");
        assert!(decoded.get_field_by_name("not_in_proto").is_none());
    }

    #[test]
    fn keeps_envelope_host_and_auction_owner_strings() {
        let catalog = Catalog::load();
        let raw = serde_json::json!({
            "envelope": {
                "event_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
                "event_name": "ad_request_started",
                "event_ts": 1710000000123i64,
                "host": "publisher",
                "auction_owner": "bidon"
            }
        });
        let encoded = catalog
            .encode_event(&raw, "chess_2000", "com.demo.chessmaster", "session")
            .expect("encode");
        let decoded = catalog
            .decode(&encoded.message_type, &encoded.payload)
            .expect("decode");
        let envelope = match decoded.get_field_by_name("envelope").unwrap().into_owned() {
            Value::Message(envelope) => envelope,
            other => panic!("envelope: {other:?}"),
        };
        assert_eq!(field_string(&envelope, "host"), "publisher");
        assert_eq!(field_string(&envelope, "auction_owner"), "bidon");
    }

    #[test]
    fn keeps_sdk_crashed_exception_list() {
        let catalog = Catalog::load();
        let raw = serde_json::json!({
            "envelope": {
                "event_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
                "event_name": "sdk_crashed",
                "event_ts": 1710000000123i64
            },
            "crashed_on_main_thread": true,
            "exceptions": [{
                "class": "java.lang.IllegalStateException",
                "frames": ["org.bidon.sdk.Session.tick"],
                "omitted_frames": 4
            }]
        });
        let encoded = catalog
            .encode_event(&raw, "chess_2000", "com.demo.chessmaster", "session")
            .expect("encode");
        let decoded = catalog
            .decode(&encoded.message_type, &encoded.payload)
            .expect("decode");
        match decoded.get_field_by_name("crashed_on_main_thread").unwrap().as_ref() {
            Value::Bool(true) => {}
            other => panic!("crashed_on_main_thread: {other:?}"),
        }
        let exceptions = decoded.get_field_by_name("exceptions").unwrap();
        let exceptions = match exceptions.as_ref() {
            Value::List(items) => items,
            other => panic!("exceptions: {other:?}"),
        };
        let exception = match &exceptions[0] {
            Value::Message(message) => message,
            other => panic!("exception: {other:?}"),
        };
        assert_eq!(field_string(exception, "class"), "java.lang.IllegalStateException");
        match exception.get_field_by_name("frames").unwrap().as_ref() {
            Value::List(frames) => match &frames[0] {
                Value::String(frame) => assert_eq!(frame, "org.bidon.sdk.Session.tick"),
                other => panic!("frame: {other:?}"),
            },
            other => panic!("frames: {other:?}"),
        }
        match exception.get_field_by_name("omitted_frames").unwrap().as_ref() {
            Value::I32(4) => {}
            other => panic!("omitted_frames: {other:?}"),
        }
    }

    fn field_string(msg: &DynamicMessage, name: &str) -> String {
        match msg.get_field_by_name(name).unwrap().as_ref() {
            Value::String(value) => value.clone(),
            other => panic!("{name}: {other:?}"),
        }
    }

    fn field_i64(msg: &DynamicMessage, name: &str) -> i64 {
        match msg.get_field_by_name(name).unwrap().as_ref() {
            Value::I64(value) => *value,
            other => panic!("{name}: {other:?}"),
        }
    }
}
