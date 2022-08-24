class LineItem < ApplicationRecord
  many_to_one :demand_source_account, key: :account_id
end
