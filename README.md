# Remote Sensors Server

## Config

```yaml
server:
  # the http server port
  port: 8080 
manager:
  # fields are the fields which the device will record, anything
  # not in these lists will be ignored
  fields:
    # define the numeric values for the device
    metrics:
      - cpu_usage
      - cpu_temp
      - cpu_clock
      - gpu_usage
      - gpu_temp
      - gpu_clock
      - fps
    # metadata are string fields which might be useful
    metadata:
      - theme

# consumers defines all of the consumers. currently, only MQTT is supported
consumers:
  mqtt:
    brokerHost: 
    brokerPort: 1883
    # topics defines a list of MQTT topics to subscribe to. the topic MUST include "{DEVICE}"
    # where the value will be used as the device ID in the manager
    topics:
      EXAMPLE/{DEVICE}/TOPIC:
        # values are mapped by "incoming field path" -> manager.fields.metrics.field
        metrics:
          "cpu temperature.last": "cpu_temp"
          "cpu usage.last": "cpu_usage"
          "fps.last": "fps"
        # values are mapped by "incoming field path" -> manager.fields.metadata.field
        metadata:
          "theme": "theme"
```