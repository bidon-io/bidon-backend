class AddPasswordToUsers < ActiveRecord::Migration[7.0]
  def change
    add_column :users, :password, :string
    User.update_all(password: 'password') # rubocop:disable Rails/SkipsModelValidations
    change_column_null :users, :password, false
  end
end
