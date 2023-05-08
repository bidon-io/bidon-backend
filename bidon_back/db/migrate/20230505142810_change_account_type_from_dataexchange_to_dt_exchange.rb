class ChangeAccountTypeFromDataexchangeToDtExchange < ActiveRecord::Migration[7.0]
  def up
    execute(
      <<~SQL.squish,
        UPDATE demand_source_accounts
        SET type = 'DemandSourceAccount::DTExchange'
        WHERE type = 'DemandSourceAccount::DataExchange'
      SQL
    )
    execute(
      <<~SQL.squish,
        UPDATE app_demand_profiles
        SET account_type = 'DemandSourceAccount::DTExchange'
        WHERE account_type = 'DemandSourceAccount::DataExchange'
      SQL
    )
  end

  def down; end
end
