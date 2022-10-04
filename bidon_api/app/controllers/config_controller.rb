# frozen_string_literal: true

class ConfigController < ApplicationController
  def create
    config_response = Api::Config::Response.new(api_request)

    render json: config_response.body, status: :ok
  end

  private

  def schema_path
    Pathname.new(Rails.root.join('json_schema', 'config.json'))
  end
end
