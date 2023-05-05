class DemandSourceAccount::DTExchange < DemandSourceAccount
  def slug
    "dtexchange_account_#{id}"
  end
end
