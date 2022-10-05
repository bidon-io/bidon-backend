# frozen_string_literal: true

require 'rails_helper'

RSpec.describe ConfigController, type: :controller do
  let(:request_params) do
    {
      device:   {
        ua:              'User Agent',
        make:            'Apple',
        model:           'iPhone',
        os:              'iOS',
        osv:             '15.0.0',
        hwv:             '14,2',
        h:               2532,
        w:               1170,
        ppi:             2,
        pxratio:         3.0,
        js:              1,
        language:        'en',
        carrier:         'Orange',
        mccmnc:          '210-102',
        connection_type: 'WIFI',
      },
      session:  {
        id:                           '51acc730-1402-11ed-861d-0242ac120002',
        launch_ts:                    '1659571550',
        launch_monotonic_ts:          '1203445',
        start_ts:                     1_659_571_550,
        monotonic_start_ts:           1_203_445,
        ts:                           1_659_571_594,
        monotonic_ts:                 1_203_497,
        memory_warnings_ts:           [
          1_659_571_572,
        ],
        memory_warnings_monotonic_ts: [
          1_203_464,
        ],
        ram_used:                     102_858_752,
        ram_size:                     5_971_034_112,
        storage_free:                 16_699_088_896,
        storage_used:                 111_182_376_960,
        battery:                      0.86,
        cpu_usage:                    0.24,
      },
      app:      {
        bundle:            'myamazing.app.com',
        key:               'some key',
        framework:         'unity',
        version:           '1.2.3',
        framework_version: '14.3.2',
        plugin_version:    '1.2.3',
      },
      user:     {
        idfa:                          'UUID',
        tracking_authorization_status: 3,
        idfv:                          'UUID',
        idg:                           'UUID',
        consent:                       {
          key1: 'value1',
        },
        coppa:                         false,
      },
      geo:      {
        lat:       23.12,
        lon:       -45.95,
        accuracy:  10,
        lastfix:   23,
        country:   'PL',
        city:      'Warsaw',
        zip:       '02-235',
        utcoffset: -432_000,
      },
      adapters: {
        admob:      {
          version:     '0.1.0.2',
          sdk_version: '7.9.0',
        },
        bidmachine: {
          version:     '0.1.0.2',
          sdk_version: '7.9.0',
        },
        applovin:   {
          version:     '0.1.0.2',
          sdk_version: '7.9.0',
        },
        appsflyer:  {
          version:     '0.1.0.2',
          sdk_version: '7.9.0',
        },
      },
      ext:      {
        key1: 'value1',
      },
      token:    '{}',
    }
  end

  context 'missing X-BidOn-Version header' do
    let(:expected_response) do
      {
        error: {
          code:    422,
          message: 'Request should contain X-BidOn-Version header',
        },
      }.to_json
    end

    it 'returns 422 with error' do
      post :create, params: request_params

      expect(response).to have_http_status(:unprocessable_entity)
      expect(response.body).to eq expected_response
    end
  end

  context 'X-BidOn-Version header present' do
    before do
      request.headers['X-BidOn-Version'] = '1.2.3'
    end

    context 'valid response' do
      let(:expected_response) do
        {
          'init'       => {
            'tmax'     => 5000,
            'adapters' => {},
          },
          'placements' => [],
          'token'      => '{}',
          'segment_id' => '',
        }.to_json
      end

      it 'returns 200 with ok' do
        allow_any_instance_of(Api::Request).to receive(:valid?).and_return(true)

        post :create, params: request_params

        expect(response).to have_http_status(:ok)
        expect(response.body).to eq expected_response
      end
    end

    context 'invalid request' do
      let(:expected_response) do
        {
          error: {
            code:    422,
            message: 'App key is invalid',
          },
        }.to_json
      end

      it 'returns 422 with error' do
        allow_any_instance_of(Api::Request).to receive(:valid?).and_return(false)

        post :create, params: request_params

        expect(response).to have_http_status(:unprocessable_entity)
        expect(response.body).to eq expected_response
      end
    end

    context 'error request' do
      let(:expected_response) do
        {
          error: {
            code:    500,
            message: 'Internal Server Error',
          },
        }.to_json
      end

      it 'returns 500 with error' do
        allow(Api::Request).to receive(:new).and_raise(StandardError)

        post :create, params: request_params

        expect(response).to have_http_status(:internal_server_error)
        expect(response.body).to eq expected_response
      end
    end
  end
end
