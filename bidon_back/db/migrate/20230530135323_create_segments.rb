class CreateSegments < ActiveRecord::Migration[7.0]
  def change
    create_table :segments do |t|
      t.string :name
      t.text :description
      t.jsonb :filters, default: []
      t.boolean :enabled, null: false, default: true
      t.references :app, null: false, foreign_key: true

      t.timestamps
    end
  end
end
