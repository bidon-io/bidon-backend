# frozen_string_literal: true

class StatsController < ApplicationController
  def create
    binding.irb
    schemer.validate(permitted_params)

    render_empty_result
  end

  private

  def schemer
    schema = Pathname.new(Rails.root.join('json_schema', 'stats.json'))
    JSONSchemer.schema(schema)
  end
end
