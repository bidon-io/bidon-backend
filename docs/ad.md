# Specs

https://github.com/prebid/openrtb

3.1 Object Model
  
    ... requires at least one of Banner (which may allow multiple formats), Video, Audio 
    and Native to define the type of impression

IAB terminology is mostly focused on web, banner = static, video = video.


We define an auction of type Banner, Interstitial, Rewarded.
In the case of a Banner, the line item has Ad Format with Adaptive Banner, Banner, LeaderBoard, MREC.

Ad type in-app UX behavior:
- Banner: controlled by the app, persistent, mostly static images
- Interstitial: full screen closable after 5 sec, controlled by the app, can be a static image, a video,
  or a playable
- Rewarded: full screen, user controlled, user needs to wait the time to finish before they can close it
  (they could close it before but then they won’t get their in-app reward)

# 
