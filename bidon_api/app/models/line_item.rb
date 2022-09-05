class LineItem < Sequel::Model
  plugin :enum

  many_to_one :demand_source_account, key: :account_id

  AD_TYPES = { interstitial: 0, banner: 1, video: 2, native: 3, mrec: 4, rewarded_video: 5 }.freeze

  enum :ad_type, AD_TYPES
end
