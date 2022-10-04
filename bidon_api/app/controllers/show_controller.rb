# frozen_string_literal: true

class ShowController < ApplicationController
  def create
    render_empty_result
  end

  private

  def schema_path
    Pathname.new(Rails.root.join('json_schema', 'show.json'))
  end
end
