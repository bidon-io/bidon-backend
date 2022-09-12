# frozen_string_literal: true

class StatsController < ApplicationController
  before_action :validate_request_schema!

  def create
    render_empty_result
  end

  private

  def schema_path
    Pathname.new(Rails.root.join('json_schema', 'stats.json'))
  end
end
