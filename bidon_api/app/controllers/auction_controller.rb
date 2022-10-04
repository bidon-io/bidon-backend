# frozen_string_literal: true

class AuctionController < ApplicationController
  def create
    auction_response = Api::Auction::Response.new(api_request)

    if auction_response.present?
      render json: auction_response.body, status: :ok
    else
      render json:   { error: { code: 422, message: 'No ads found' } },
             status: :unprocessable_entity
    end
  end

  private

  def schema_path
    Pathname.new(Rails.root.join('json_schema', 'auction.json'))
  end
end
