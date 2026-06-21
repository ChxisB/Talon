declare global {
  const TALON_VERSION: string
  const TALON_CHANNEL: string
}

export const InstallationVersion = typeof TALON_VERSION === "string" ? TALON_VERSION : "local"
export const InstallationChannel = typeof TALON_CHANNEL === "string" ? TALON_CHANNEL : "local"
export const InstallationLocal = InstallationChannel === "local"
