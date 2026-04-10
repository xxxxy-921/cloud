/** Parse a User-Agent string into a human-readable device description. */
export function parseUserAgent(ua: string): string {
  if (!ua) return "-"

  let browser = "Unknown"
  let os = "Unknown"

  // Browser detection
  if (/Edg\//i.test(ua)) {
    browser = "Edge"
  } else if (/Chrome\//i.test(ua) && !/Chromium/i.test(ua)) {
    browser = "Chrome"
  } else if (/Firefox\//i.test(ua)) {
    browser = "Firefox"
  } else if (/Safari\//i.test(ua) && !/Chrome/i.test(ua)) {
    browser = "Safari"
  } else if (/OPR\//i.test(ua) || /Opera/i.test(ua)) {
    browser = "Opera"
  }

  // OS detection
  if (/Windows/i.test(ua)) {
    os = "Windows"
  } else if (/Mac OS X|macOS/i.test(ua)) {
    os = "macOS"
  } else if (/Linux/i.test(ua) && !/Android/i.test(ua)) {
    os = "Linux"
  } else if (/Android/i.test(ua)) {
    os = "Android"
  } else if (/iPhone|iPad|iPod/i.test(ua)) {
    os = "iOS"
  }

  return `${browser} / ${os}`
}
