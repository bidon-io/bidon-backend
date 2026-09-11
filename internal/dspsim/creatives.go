package dspsim

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/bidon-io/bidon-backend/internal/ad"
)

//go:embed default_creatives.json
var defaultCreatives []byte

// Creative types shipped by default. The loader treats the key as opaque, so a
// new type is a JSON edit; only the format eligibility map below is Go.
const (
	TypeStaticBanner      = "static_banner"
	TypeMRAIDBanner       = "mraid_banner"
	TypeMRAIDInterstitial = "mraid_interstitial"
	TypeVASTVideo         = "vast_video"
)

// DefaultDSPBucket is the fallback bucket used when the requesting DSP has no
// bucket of its own, or no eligible creative in it.
const DefaultDSPBucket = "default"

// eligibleTypes maps a matched ad format onto the creative types that can serve
// it, in preference order.
var eligibleTypes = map[ad.Format][]string{
	ad.BannerFormat:      {TypeStaticBanner, TypeMRAIDBanner},
	ad.LeaderboardFormat: {TypeStaticBanner, TypeMRAIDBanner},
	ad.MRECFormat:        {TypeStaticBanner, TypeMRAIDBanner},
	ad.AdaptiveFormat:    {TypeStaticBanner, TypeMRAIDBanner},
}

// fullscreenTypes are used when no banner format applies (interstitial and
// rewarded impressions).
var fullscreenTypes = map[ad.Type][]string{
	ad.InterstitialType: {TypeMRAIDInterstitial, TypeVASTVideo, TypeStaticBanner},
	ad.RewardedType:     {TypeVASTVideo, TypeMRAIDInterstitial},
}

// Creative is one entry of the creative library.
type Creative struct {
	ID      string      `json:"id"`
	Width   int64       `json:"w,omitempty"`
	Height  int64       `json:"h,omitempty"`
	Formats []ad.Format `json:"formats"`
	AdTypes []ad.Type   `json:"ad_types,omitempty"`
	ADomain []string    `json:"adomain,omitempty"`
	Bundle  string      `json:"bundle,omitempty"`
	CampID  string      `json:"cid,omitempty"`
	Weight  int         `json:"weight,omitempty"`
	ADM     string      `json:"adm"`

	// Type and DSP are filled in by the loader from the position in the index.
	Type string `json:"type"`
	DSP  string `json:"dsp"`

	tmpl *template.Template
}

// CreativeData is the template context available inside an "adm".
type CreativeData struct {
	BidID         string
	ImpID         string
	RequestID     string
	DSP           string
	CreativeID    string
	Currency      string
	Price         float64
	W             int64
	H             int64
	PublicURL     string
	AssetURL      string
	ClickURL      string
	ImpressionURL string
	TrackURL      string
}

// Render expands the creative markup for one bid.
func (c *Creative) Render(data CreativeData) (string, error) {
	var sb strings.Builder
	if err := c.tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("render creative %q: %w", c.ID, err)
	}
	return sb.String(), nil
}

func (c *Creative) servesFormat(format ad.Format, adType ad.Type) bool {
	if format != ad.EmptyFormat {
		for _, f := range c.Formats {
			if f == format || formatsInterchangeable(f, format) {
				return true
			}
		}
		return false
	}
	// Fullscreen: match on ad type when the creative declares one, otherwise
	// accept any creative of an eligible type.
	if len(c.AdTypes) == 0 {
		return true
	}
	for _, t := range c.AdTypes {
		if t == adType {
			return true
		}
	}
	return false
}

// fitsSize rejects a banner creative larger than the requested slot. Creatives
// without a declared size (fullscreen, video) always fit.
func (c *Creative) fitsSize(w, h int64) bool {
	if c.Width == 0 || c.Height == 0 || w == 0 || h == 0 {
		return true
	}
	return c.Width <= w && c.Height <= h
}

// Library is the DSP -> creative type -> creatives index.
type Library struct {
	// Source records where the library was loaded from, for /debug/creatives.
	Source string

	dsps map[string]map[string][]*Creative

	mu     sync.Mutex
	served map[string]int64
}

// rawLibrary is the on-disk shape: {dsp: {creative_type: [creative, ...]}}.
type rawLibrary map[string]map[string][]*Creative

// LoadLibrary reads a creative library from path, or the embedded default
// library when path is empty. A malformed library is an error, so a bad file
// fails startup rather than a bid.
func LoadLibrary(path string) (*Library, error) {
	raw := defaultCreatives
	source := "embedded"

	if path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read creatives file %q: %w", path, err)
		}
		raw = content
		source = path
	}

	var parsed rawLibrary
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse creatives %s: %w", source, err)
	}

	lib := &Library{
		Source: source,
		dsps:   map[string]map[string][]*Creative{},
		served: map[string]int64{},
	}

	for dsp, byType := range parsed {
		dspKey := strings.ToLower(dsp)
		seen := map[string]bool{}
		bucket := map[string][]*Creative{}

		for creativeType, creatives := range byType {
			for _, creative := range creatives {
				if err := prepareCreative(creative, dspKey, creativeType, seen); err != nil {
					return nil, err
				}
				bucket[creativeType] = append(bucket[creativeType], creative)
			}
			sort.Slice(bucket[creativeType], func(i, j int) bool {
				return bucket[creativeType][i].ID < bucket[creativeType][j].ID
			})
		}
		lib.dsps[dspKey] = bucket
	}

	if _, ok := lib.dsps[DefaultDSPBucket]; !ok {
		return nil, fmt.Errorf("creatives %s: missing %q bucket", source, DefaultDSPBucket)
	}

	return lib, nil
}

func prepareCreative(c *Creative, dsp, creativeType string, seen map[string]bool) error {
	if c.ID == "" {
		return fmt.Errorf("creative in %s/%s has no id", dsp, creativeType)
	}
	if seen[c.ID] {
		return fmt.Errorf("duplicate creative id %q in dsp %q", c.ID, dsp)
	}
	seen[c.ID] = true

	if strings.TrimSpace(c.ADM) == "" {
		return fmt.Errorf("creative %q has empty adm", c.ID)
	}
	if c.Weight < 0 {
		return fmt.Errorf("creative %q has negative weight %d", c.ID, c.Weight)
	}
	if c.Weight == 0 {
		c.Weight = 1
	}

	for _, f := range c.Formats {
		if !knownFormat(f) {
			return fmt.Errorf("creative %q declares unknown format %q", c.ID, f)
		}
	}
	for _, t := range c.AdTypes {
		if t != ad.BannerType && t != ad.InterstitialType && t != ad.RewardedType {
			return fmt.Errorf("creative %q declares unknown ad type %q", c.ID, t)
		}
	}
	if len(c.Formats) == 0 && len(c.AdTypes) == 0 {
		return fmt.Errorf("creative %q declares neither formats nor ad_types", c.ID)
	}

	tmpl, err := template.New(c.ID).Parse(c.ADM)
	if err != nil {
		return fmt.Errorf("creative %q has an invalid adm template: %w", c.ID, err)
	}

	c.tmpl = tmpl
	c.Type = creativeType
	c.DSP = dsp
	if c.CampID == "" {
		c.CampID = "camp_" + c.ID
	}
	return nil
}

func knownFormat(f ad.Format) bool {
	switch f {
	case ad.BannerFormat, ad.LeaderboardFormat, ad.MRECFormat, ad.AdaptiveFormat:
		return true
	default:
		return false
	}
}

// Selection is the outcome of picking a creative.
type Selection struct {
	Creative *Creative
	// Bucket is the DSP bucket the creative came from, which differs from the
	// requested DSP when the default bucket was used as a fallback.
	Bucket string
	// FellBack reports whether the requested DSP bucket could not serve.
	FellBack bool
}

// Select picks a creative for the requested DSP and matched ad slot. forcedID,
// when set, pins a creative by id and bypasses the weighted draw.
func (l *Library) Select(dsp string, adType ad.Type, format ad.Format, w, h int64, forcedID string, rnd *rand.Rand) (*Selection, bool) {
	dsp = strings.ToLower(strings.TrimSpace(dsp))
	if dsp == "" {
		dsp = DefaultDSPBucket
	}

	if forcedID != "" {
		if c := l.find(dsp, forcedID); c != nil {
			return &Selection{Creative: c, Bucket: dsp}, true
		}
		if c := l.find(DefaultDSPBucket, forcedID); c != nil {
			return &Selection{Creative: c, Bucket: DefaultDSPBucket, FellBack: dsp != DefaultDSPBucket}, true
		}
		return nil, false
	}

	types := typesFor(adType, format)

	if c := l.draw(dsp, types, adType, format, w, h, rnd); c != nil {
		return &Selection{Creative: c, Bucket: dsp}, true
	}
	if dsp != DefaultDSPBucket {
		if c := l.draw(DefaultDSPBucket, types, adType, format, w, h, rnd); c != nil {
			return &Selection{Creative: c, Bucket: DefaultDSPBucket, FellBack: true}, true
		}
	}
	return nil, false
}

func typesFor(adType ad.Type, format ad.Format) []string {
	if types, ok := eligibleTypes[format]; ok {
		return types
	}
	if types, ok := fullscreenTypes[adType]; ok {
		return types
	}
	return []string{TypeStaticBanner, TypeMRAIDBanner, TypeMRAIDInterstitial, TypeVASTVideo}
}

// draw collects every eligible creative across the candidate types and picks
// one with probability proportional to its weight.
func (l *Library) draw(dsp string, types []string, adType ad.Type, format ad.Format, w, h int64, rnd *rand.Rand) *Creative {
	bucket, ok := l.dsps[dsp]
	if !ok {
		return nil
	}

	var eligible []*Creative
	total := 0
	for _, creativeType := range types {
		for _, c := range bucket[creativeType] {
			if !c.servesFormat(format, adType) || !c.fitsSize(w, h) {
				continue
			}
			eligible = append(eligible, c)
			total += c.Weight
		}
	}
	if total == 0 {
		return nil
	}

	pick := rnd.Intn(total)
	for _, c := range eligible {
		pick -= c.Weight
		if pick < 0 {
			return c
		}
	}
	return eligible[len(eligible)-1]
}

func (l *Library) find(dsp, id string) *Creative {
	for _, creatives := range l.dsps[dsp] {
		for _, c := range creatives {
			if c.ID == id {
				return c
			}
		}
	}
	return nil
}

// MarkServed increments the serve counter of a creative.
func (l *Library) MarkServed(bucket, id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.served[bucket+"/"+id]++
}

// Buckets returns the DSP bucket names in the library.
func (l *Library) Buckets() []string {
	out := make([]string, 0, len(l.dsps))
	for dsp := range l.dsps {
		out = append(out, dsp)
	}
	sort.Strings(out)
	return out
}

// CreativeSummary is the /debug/creatives view of one creative.
type CreativeSummary struct {
	ID      string      `json:"id"`
	DSP     string      `json:"dsp"`
	Type    string      `json:"type"`
	Formats []ad.Format `json:"formats,omitempty"`
	AdTypes []ad.Type   `json:"ad_types,omitempty"`
	Width   int64       `json:"w,omitempty"`
	Height  int64       `json:"h,omitempty"`
	Weight  int         `json:"weight"`
	Served  int64       `json:"served"`
}

// Describe returns the library contents for one DSP bucket, or all of them when
// dsp is empty.
func (l *Library) Describe(dsp string) map[string][]CreativeSummary {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := map[string][]CreativeSummary{}
	for bucket, byType := range l.dsps {
		if dsp != "" && bucket != strings.ToLower(dsp) {
			continue
		}
		for _, creatives := range byType {
			for _, c := range creatives {
				out[bucket] = append(out[bucket], CreativeSummary{
					ID:      c.ID,
					DSP:     bucket,
					Type:    c.Type,
					Formats: c.Formats,
					AdTypes: c.AdTypes,
					Width:   c.Width,
					Height:  c.Height,
					Weight:  c.Weight,
					Served:  l.served[bucket+"/"+c.ID],
				})
			}
		}
		sort.Slice(out[bucket], func(i, j int) bool { return out[bucket][i].ID < out[bucket][j].ID })
	}
	return out
}
