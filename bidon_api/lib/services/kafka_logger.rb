module KafkaLogger
  module_function

  def log_click(event)
    $kafka_producer.produce(JSON.dump(event), topic: ENV.fetch('KAFKA_CLICK_TOPIC'))
  end

  def log_finish(event)
    $kafka_producer.produce(JSON.dump(event), topic: ENV.fetch('KAFKA_FINISH_TOPIC'))
  end

  def log_show(event)
    $kafka_producer.produce(JSON.dump(event), topic: ENV.fetch('KAFKA_SHOW_TOPIC'))
  end

  def log_stats(event)
    $kafka_producer.produce(JSON.dump(event), topic: ENV.fetch('KAFKA_STATS_TOPIC'))
  end
end
