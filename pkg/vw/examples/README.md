# Examples

Reasonable defaults have been chosen, but if these examples do not work then you'll likely need to edit the definition of the video and data sources to suit the equipment you have.

## Video

Run `vw-rule-video` to set up a rule for the video

Start `ffmpeg-camera` to stream live video to `vw` (leave running)

## Data

Run `vw-rule-data` to set up a rule for the data 

In separate terminals:

  - start `websocat-data` to provide a bidirection tcp to websocket bridge
  - start `socat-data` to bridge the serial port to tcp port of `websocat`








Stream mpeg1video 


socat-data
vw-rules
vw-rules-video-only
websocat-data
