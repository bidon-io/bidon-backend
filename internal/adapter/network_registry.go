package adapter

// networks is the canonical network catalog (key, label, account type, auction
// membership, and processed-config field projections).
var networks = []Network{
	{
		Key:             AdikteevKey,
		Label:           "Adikteev",
		AccountType:     "DemandSourceAccount::Adikteev",
		SupportsBidding: true,
		AccountExtra:    []FieldMap{CopyKey("sdk_instance_id")},
	},
	{
		Key:               AdmobKey,
		Label:             "AdMob",
		AccountType:       "DemandSourceAccount::Admob",
		SupportsWaterfall: true,
	},
	{
		Key:             AmazonKey,
		Label:           "Amazon",
		AccountType:     "DemandSourceAccount::Amazon",
		SupportsBidding: true,
		AccountExtra:    []FieldMap{CopyKey("price_points_map")},
	},
	{
		Key:               ApplovinKey,
		Label:             "AppLovin",
		AccountType:       "DemandSourceAccount::Applovin",
		SupportsWaterfall: true,
	},
	{
		Key:               BidmachineKey,
		Label:             "BidMachine",
		AccountType:       "DemandSourceAccount::BidMachine",
		SupportsBidding:   true,
		SupportsWaterfall: true,
		AccountExtra: []FieldMap{
			CopyKey("seller_id"),
			CopyKey("endpoint"),
			CopyKey("mediation_config"),
		},
	},
	{
		Key:               BigoAdsKey,
		Label:             "Bigo Ads",
		AccountType:       "DemandSourceAccount::BigoAds",
		SupportsBidding:   true,
		SupportsWaterfall: true,
		AccountExtra:      []FieldMap{RenameKey("publisher_id", "seller_id")},
		AppData:           []FieldMap{CopyKey("app_id")},
		AdUnitExtra: []FieldMap{
			RenameKey("slot_id", "tag_id"),
			CopyKey("placement_id"),
		},
	},
	{
		Key:               ChartboostKey,
		Label:             "Chartboost",
		AccountType:       "DemandSourceAccount::Chartboost",
		SupportsWaterfall: true,
	},
	{
		Key:               DTExchangeKey,
		Label:             "DT Exchange",
		AccountType:       "DemandSourceAccount::DtExchange",
		SupportsWaterfall: true,
	},
	{
		Key:               GAMKey,
		Label:             "Google Ad Manager",
		AccountType:       "DemandSourceAccount::GAM",
		SupportsWaterfall: true,
	},
	{
		Key:             InmobiKey,
		Label:           "InMobi",
		AccountType:     "DemandSourceAccount::Inmobi",
		SupportsBidding: true,
		AppData:         []FieldMap{RenameKey("app_key", "app_id")},
		AdUnitExtra:     []FieldMap{CopyKey("placement_id")},
	},
	{
		Key:               IronSourceKey,
		Label:             "IronSource",
		AccountType:       "DemandSourceAccount::IronSource",
		SupportsWaterfall: true,
	},
	{
		Key:             MetaKey,
		Label:           "Meta",
		AccountType:     "DemandSourceAccount::Meta",
		SupportsBidding: true,
		AppData:          []FieldMap{CopyKey("app_id")},
		AdUnitExtra:      []FieldMap{RenameKey("placement_id", "tag_id")},
		InjectEnvSecrets: injectMetaEnvSecrets,
	},
	{
		Key:               MintegralKey,
		Label:             "Mintegral",
		AccountType:       "DemandSourceAccount::Mintegral",
		SupportsBidding:   true,
		SupportsWaterfall: true,
		AccountExtra:      []FieldMap{RenameKey("publisher_id", "seller_id")},
		AppData:           []FieldMap{CopyKey("app_id")},
		AdUnitExtra: []FieldMap{
			RenameKey("unit_id", "tag_id"),
			CopyKey("placement_id"),
		},
	},
	{
		Key:             MobileFuseKey,
		Label:           "MobileFuse",
		AccountType:     "DemandSourceAccount::MobileFuse",
		SupportsBidding: true,
		AdUnitExtra:     []FieldMap{RenameKey("placement_id", "tag_id")},
	},
	{
		Key:             MolocoKey,
		Label:           "Moloco",
		AccountType:     "DemandSourceAccount::Moloco",
		SupportsBidding: true,
		AppData:          []FieldMap{RenameKey("app_key", "app_id")},
		AdUnitExtra:      []FieldMap{RenameKey("ad_unit_id", "tag_id")},
		InjectEnvSecrets: injectMolocoEnvSecrets,
	},
	{
		Key:             StartIOKey,
		Label:           "Start.io",
		AccountType:     "DemandSourceAccount::StartIO",
		SupportsBidding: true,
		AccountExtra:    []FieldMap{CopyKey("account")},
		AppData:         []FieldMap{CopyKey("app_id")},
		AdUnitExtra:     []FieldMap{CopyKey("tag_id")},
	},
	{
		Key:             TaurusXKey,
		Label:           "TaurusX",
		AccountType:     "DemandSourceAccount::TaurusX",
		SupportsBidding: true,
		AppData:         []FieldMap{CopyKey("app_id")},
		AdUnitExtra:     []FieldMap{RenameKey("placement_id", "tag_id")},
	},
	{
		Key:               UnityAdsKey,
		Label:             "Unity Ads",
		AccountType:       "DemandSourceAccount::UnityAds",
		SupportsWaterfall: true,
	},
	{
		Key:               VKAdsKey,
		Label:             "VK Ads",
		AccountType:       "DemandSourceAccount::VKAds",
		SupportsBidding:   true,
		SupportsWaterfall: true,
		AppData:           []FieldMap{CopyKey("app_id")},
		AdUnitExtra:       []FieldMap{RenameKey("slot_id", "tag_id")},
	},
	{
		Key:               VungleKey,
		Label:             "Vungle",
		AccountType:       "DemandSourceAccount::Vungle",
		SupportsBidding:   true,
		SupportsWaterfall: true,
		AccountExtra:      []FieldMap{RenameKey("account_id", "seller_id")},
		AppData:           []FieldMap{CopyKey("app_id")},
		AdUnitExtra:       []FieldMap{RenameKey("placement_id", "tag_id")},
	},
	{
		Key:               YandexKey,
		Label:             "Yandex",
		AccountType:       "DemandSourceAccount::Yandex",
		SupportsBidding:   true,
		SupportsWaterfall: true,
		AdUnitExtra:       []FieldMap{CopyKey("ad_unit_id")},
	},
	{
		Key:             ZmaticooKey,
		Label:           "Zmaticoo",
		AccountType:     "DemandSourceAccount::Zmaticoo",
		SupportsBidding: true,
		AppData:         []FieldMap{RenameKey("app_key", "app_id")},
		AdUnitExtra:     []FieldMap{CopyKey("placement_id")},
	},
}

var networksByKey map[Key]Network

func init() {
	networksByKey = make(map[Key]Network, len(networks))
	for _, n := range networks {
		networksByKey[n.Key] = n
	}
}

// Networks returns the full catalog in registration order.
func Networks() []Network {
	out := make([]Network, len(networks))
	copy(out, networks)
	return out
}

// NetworkByKey looks up a network by adapter key.
func NetworkByKey(key Key) (Network, bool) {
	n, ok := networksByKey[key]
	return n, ok
}

// BiddingKeys returns keys with SupportsBidding.
func BiddingKeys() []Key {
	keys := make([]Key, 0, len(networks))
	for _, n := range networks {
		if n.SupportsBidding {
			keys = append(keys, n.Key)
		}
	}
	return keys
}

// WaterfallKeys returns keys with SupportsWaterfall.
func WaterfallKeys() []Key {
	keys := make([]Key, 0, len(networks))
	for _, n := range networks {
		if n.SupportsWaterfall {
			keys = append(keys, n.Key)
		}
	}
	return keys
}
