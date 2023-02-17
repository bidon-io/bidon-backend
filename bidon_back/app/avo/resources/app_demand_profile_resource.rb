class AppDemandProfileResource < Avo::BaseResource
  self.title = :id

  field :id, as: :id
  field :app, as: :belongs_to, required: true
  field :demand_source, as: :belongs_to, required: true
  field :account, as: :belongs_to, required: true
  field :data, as: :code
  field :account_type,
        as:       :select,
        required: true,
        options:  {
          BidmachineAccount: 'DemandSourceAccount::BidMachine',
          AdmobAccount:      'DemandSourceAccount::Admob',
          ApplovinAccount:   'DemandSourceAccount::Applovin',
        }
end
