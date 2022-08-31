class Country < Sequel::Model
  def self.find_cached(code)
    Cache.global.fetch("country_#{code}") do
      Country.find_by(code:) || Country.find_by(code: 'ZZ')
    end
  end
end
