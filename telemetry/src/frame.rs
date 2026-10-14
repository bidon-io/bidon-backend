/// Confluent protobuf wire format used by the Go telemetry logger:
/// magic `0x00`, big-endian schema id, zigzag varint message indexes, payload.
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
    write_zigzag(&mut out, indexes.len() as u64);
    for index in indexes {
        write_zigzag(&mut out, *index);
    }
    out
}

fn write_zigzag(out: &mut Vec<u8>, value: u64) {
    // Confluent writes the index length and each index with zigzag varints,
    // the same encoding as encoding/binary.AppendVarint.
    write_varint(out, value << 1);
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
    fn nonzero_index_is_zigzag_length_then_index() {
        // length 1 -> zigzag 2, index 4 -> zigzag 8
        assert_eq!(encode_indexes(&[4]), vec![2, 8]);
    }

    #[test]
    fn index_14_matches_confluent_zigzag() {
        // AdRequestStarted. length 1 -> 2, index 14 -> 28.
        assert_eq!(encode_indexes(&[14]), vec![0x02, 0x1c]);
    }

    #[test]
    fn zigzag_spans_two_bytes_past_63() {
        // index 64 zigzags to 128, which is a two-byte unsigned varint.
        assert_eq!(encode_indexes(&[64]), vec![0x02, 0x80, 0x01]);
    }

    #[test]
    fn frame_prefixes_magic_and_schema_id() {
        let framed = frame(7, &[4], &[0x0a]);
        assert_eq!(framed[0], 0);
        assert_eq!(&framed[1..5], &7i32.to_be_bytes());
        assert_eq!(&framed[5..], &[2, 8, 0x0a]);
    }
}
