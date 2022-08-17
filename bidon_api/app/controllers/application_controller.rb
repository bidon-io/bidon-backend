class ApplicationController < ActionController::API
  before_action :set_sentry_context
  before_action :validate_bidon_header!

  def validate_bidon_header!
    return if request.env['X-BidOn-Version'].present?

    render json:   { error: { code: 422, message: 'Request should contain X-BidOn-Version header' } },
           status: :unprocessable_entity
  end

  rescue_from StandardError do |_e|
    render json: { error: { code: 500, message: 'Internal Server Error' } }, status: :internal_server_error
  end

  def set_sentry_context
    Sentry.set_extras(params: params.to_unsafe_h, session: session.to_hash)
  end
end
