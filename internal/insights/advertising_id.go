package insights

// ResolveAdvertisingID returns the device advertising identifier used as a
// stable key component for provider state.
//
// Platform rules:
//   - iOS: IDFA, fallback to IDFV
//   - android: IDG
//   - unknown platform or missing base request: empty string
func ResolveAdvertisingID(req InitRequest) string {
	if req.BaseRequest == nil {
		return ""
	}

	switch req.BaseRequest.Device.OS {
	case "iOS":
		if req.IDFA != "" {
			return req.IDFA
		}
		return req.IDFV
	case "android":
		return req.IDG
	default:
		return ""
	}
}
