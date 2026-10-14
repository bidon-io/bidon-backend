/// Confluent protobuf wire format used by the Go telemetry logger:
/// magic `0x00`, big-endian schema id, unsigned varint message indexes, payload.
/// A single index of 0 is the special case of one zero byte.
pub fn frame(schema_id: i32, indexes: &[u64], payload: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(5 + indexes.len() + 1 + payload.len());
    out.push(0);
    out.extend_from_slice(&schema_id.to_be_bytes());
    out.extend_from_slice(&encode_indexes(indexes));
    out.extend_from_slice(payload);
    out
}

pub fn encode_indexes(indexes: &[u64]) -> Vec<u8> {
    if indexes == [0] {
        return vec![0];
    }
    let mut out = Vec::new();
    write_varint(&mut out, indexes.len() as u64);
    for index in indexes {
        write_varint(&mut out, *index);
    }
    out
}

fn write_varint(out: &mut Vec<u8>, mut value: u64) {
    loop {
        let mut byte = (value & 0x7f) as u8;
        value >>= 7;
        if value != 0 {
            byte |= 0x80;
        }
        out.push(byte);
        if value == 0 {
            break;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn zero_index_is_a_single_zero_byte() {
        assert_eq!(encode_indexes(&[0]), vec![0]);
    }

    #[test]
    fn nonzero_index_is_length_then_index() {
        assert_eq!(encode_indexes(&[4]), vec![1, 4]);
    }

    #[test]
    fn frame_prefixes_magic_and_schema_id() {
        let framed = frame(7, &[4], &[0x0a]);
        assert_eq!(framed[0], 0);
        assert_eq!(&framed[1..5], &7i32.to_be_bytes());
        assert_eq!(&framed[5..], &[1, 4, 0x0a]);
    }
}
