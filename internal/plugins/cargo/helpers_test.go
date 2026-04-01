package cargo

// buildPublishBody constructs the Cargo publish wire-format body from JSON metadata and crate bytes.
// Used only in tests to simulate crate publish requests.
func buildPublishBody(meta, crateFile []byte) []byte {
	metaLen := uint32(len(meta))
	crateLen := uint32(len(crateFile))

	buf := make([]byte, 0, 4+len(meta)+4+len(crateFile))
	buf = append(buf,
		byte(metaLen), byte(metaLen>>8), byte(metaLen>>16), byte(metaLen>>24),
	)
	buf = append(buf, meta...)
	buf = append(buf,
		byte(crateLen), byte(crateLen>>8), byte(crateLen>>16), byte(crateLen>>24),
	)
	buf = append(buf, crateFile...)
	return buf
}
