# secret.yml section for kafka
# kafka_appodeal:
#   :seed_brokers:
#     - "127.0.0.1:9093"
#     - "127.0.0.1:9094"
#     - "127.0.0.1:9095"
#   :client_id: appodeal_dev
#   :delivery_interval: 30
#   :delivery_threshold: 100

class KafkaProducerStub
  def produce(_event, _options)
    # do nothing
  end

  def shutdown
    # do nothing
  end
end

seed_brokers = ENV.fetch('KAFKA_BROKERS_LIST').split(', ')
client_id = ENV.fetch('KAFKA_CLIENT_ID')

$kafka = Kafka.new(seed_brokers:, client_id:, logger: Rails.logger)

delivery_threshold = ENV.fetch('KAFKA_DELIVERY_THRESHOLD')
delivery_interval = ENV.fetch('KAFKA_DELIVERY_INTERVAL')

if Rails.env.production?
  $kafka_producer = $kafka.async_producer(delivery_threshold:, delivery_interval:)
  at_exit { $kafka_producer.shutdown }
else
  $kafka_producer = KafkaProducerStub.new
end
