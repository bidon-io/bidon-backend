class SetFormatInAmazonLineItems < ActiveRecord::Migration[7.0]
  def up
    LineItem.where(account_type: "DemandSourceAccount::Amazon").find_each do |line_item|
      extra = line_item.extra
      case line_item.ad_type
      when 'banner'
        format = line_item.format == 'MREC' ? 'MREC' : 'BANNER'
      when 'rewarded'
        format = 'REWARDED'
      when 'interstitial'
        format = extra['is_video'] ? 'VIDEO' : 'INTERSTITIAL'
      end

      line_item.update!(extra: { slot_uuid: extra['slot_uuid'], format: format }) if !extra['format']
    end
  end

  def down
    raise ActiveRecord::IrreversibleMigration
  end
end
