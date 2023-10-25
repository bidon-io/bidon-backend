class SetFormatInAmazonLineItems < ActiveRecord::Migration[7.0]
  def up
    # set format in amazon line items
    LineItem.where(account_type: 'DemandSourceAccount::Amazon').find_each do |line_item|
      extra = line_item.extra
      case line_item.ad_type
      when 'banner'
        format = line_item.format == 'MREC' ? 'MREC' : 'BANNER'
      when 'rewarded'
        format = 'REWARDED'
      when 'interstitial'
        format = extra['is_video'] ? 'VIDEO' : 'INTERSTITIAL'
      end

      line_item.update!(extra: extra.merge({ format: })) unless extra['format']
    end

    # set placement_id in vungle, meta and mobilefuse line items
    LineItem.where(account_type: %w[DemandSourceAccount::Vungle DemandSourceAccount::Meta DemandSourceAccount::MobileFuse]).find_each do |line_item|
      line_item.update!(extra: extra.merge({ placement_id: line_item.code }))
    end
  end

  def down
    raise ActiveRecord::IrreversibleMigration
  end
end
