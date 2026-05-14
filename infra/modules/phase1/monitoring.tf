resource "digitalocean_monitor_alert" "cpu_high" {
  count = var.monitor_alert_email != null ? 1 : 0

  alerts {
    email = [var.monitor_alert_email]
  }

  compare     = "GreaterThan"
  description = "${var.name_prefix}: Droplet CPU is high"
  enabled     = true
  entities    = [local.droplet_entity]

  type   = "v1/insights/droplet/cpu"
  value  = 80
  window = "5m"
}

resource "digitalocean_monitor_alert" "memory_high" {
  count = var.monitor_alert_email != null ? 1 : 0

  alerts {
    email = [var.monitor_alert_email]
  }

  compare     = "GreaterThan"
  description = "${var.name_prefix}: Droplet memory utilization is high"
  enabled     = true
  entities    = [local.droplet_entity]

  type   = "v1/insights/droplet/memory_utilization_percent"
  value  = 85
  window = "5m"
}

resource "digitalocean_monitor_alert" "disk_high" {
  count = var.monitor_alert_email != null ? 1 : 0

  alerts {
    email = [var.monitor_alert_email]
  }

  compare     = "GreaterThan"
  description = "${var.name_prefix}: Droplet disk utilization is high"
  enabled     = true
  entities    = [local.droplet_entity]

  type   = "v1/insights/droplet/disk_utilization_percent"
  value  = 85
  window = "5m"
}
