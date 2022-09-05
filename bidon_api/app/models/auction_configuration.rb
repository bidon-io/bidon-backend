class AuctionConfiguration < Sequel::Model
  plugin :enum

  AD_TYPES = { interstitial: 1, banner: 2, rewarded_video: 3 }.freeze

  enum :ad_type, AD_TYPES
end
