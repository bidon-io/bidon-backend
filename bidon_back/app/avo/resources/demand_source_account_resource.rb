class DemandSourceAccountResource < Avo::BaseResource
  self.title = :id

  field :id, as: :id
  field :user, as: :belongs_to, required: true
  field :type,
        as:       :select,
        required: true,
        options:  {
          BidmachineAccount: 'DemandSourceAccount::BidMachine',
          AdmobAccount:      'DemandSourceAccount::Admob',
          ApplovinAccount:   'DemandSourceAccount::Applovin',
        }
  field :demand_source, as: :belongs_to, required: true
  field :bidding, as: :boolean, required: true
  field :extra, as: :code
end
