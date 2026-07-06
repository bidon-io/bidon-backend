package dspsimulator

import (
	"fmt"
	"slices"

	"github.com/prebid/openrtb/v19/openrtb2"
)

func BuildKeys(req *openrtb2.BidRequest) []string {
	var keys []string

	os := ""
	if req.Device != nil {
		os = req.Device.OS
	}

	for _, imp := range req.Imp {
		dm := imp.DisplayManager
		if dm == "" {
			continue
		}

		mediaKeys := buildMediaKeys(dm, os, &imp)
		keys = append(keys, mediaKeys...)
	}

	return keys
}

func buildMediaKeys(dm, os string, imp *openrtb2.Imp) []string {
	var keys []string

	if imp.Banner != nil {
		w := int64(0)
		h := int64(0)
		if imp.Banner.W != nil {
			w = *imp.Banner.W
		}
		if imp.Banner.H != nil {
			h = *imp.Banner.H
		}

		base := fmt.Sprintf("%s/%s/banner/%dx%d", dm, os, w, h)
		keys = append(keys, base)

		if hasMRAID(imp) {
			keys = append(keys, base+"/mraid")
		}
	}

	if imp.Video != nil {
		w := imp.Video.W
		h := imp.Video.H

		if isVAST(imp) {
			keys = append(keys, fmt.Sprintf("%s/%s/video/vast", dm, os))
		} else {
			keys = append(keys, fmt.Sprintf("%s/%s/video/%dx%d", dm, os, w, h))
		}
	}

	if imp.Native != nil {
		keys = append(keys, fmt.Sprintf("%s/%s/native", dm, os))
	}

	return keys
}

func hasMRAID(imp *openrtb2.Imp) bool {
	if imp.Banner == nil {
		return false
	}
	for _, api := range imp.Banner.API {
		if api == 3 || api == 5 {
			return true
		}
	}
	return false
}

func isVAST(imp *openrtb2.Imp) bool {
	if imp.Video == nil {
		return false
	}
	for _, mime := range imp.Video.MIMEs {
		if slices.Contains([]string{"video/mp4", "video/x-m4v", "video/quicktime", "video/mpeg", "video/avi"}, mime) {
			return true
		}
	}
	return false
}
