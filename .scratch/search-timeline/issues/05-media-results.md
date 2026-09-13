# 05: Media results

**What to build:** Make Media searches and media-bearing post results return stable image, video, and animated-GIF data through the Search command, including one directly usable playback URL rather than X's variant grammar.

**Blocked by:** 02: Top and Latest post results.

**Status:** resolved

Before starting, read the parent SearchTimeline specification, domain glossary, accepted normalized contract, and machine-readable schema. Treat them as authoritative.

- [x] Posts inside the observed Media-tab grid module become ordinary flat post results in their returned order.
- [x] Media attached to posts in any tab uses the same normalized media collection.
- [x] Upstream photos normalize to `image`, while videos and animated GIFs use their corresponding public media types.
- [x] Each media object exposes string ID, media type, URL, preview URL, width, height, alternative text, and duration in milliseconds with null for unavailable values.
- [x] Images use the image destination as their primary URL.
- [x] Videos and animated GIFs select the highest-bitrate available MP4 as their primary URL and use the image thumbnail as their preview URL.
- [x] HLS and upstream playback-variant arrays remain internal and do not appear in normalized output.
- [x] Missing bitrate, dimensions, duration, alternative text, preview, and playback URLs follow deterministic null behavior without panics or invented values.
- [x] Exact golden output covers image, video, animated-GIF, mixed-media, and empty-media posts across ordinary and Media-tab containers.
- [x] The behavior is demonstrable through `twt search --tab media` with a controlled requester and remains independent of live X.
- [x] Deterministic repository checks remain green.
