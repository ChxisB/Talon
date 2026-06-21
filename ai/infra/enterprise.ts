import { SECRET } from "./secret"
import { shortDomain } from "./stage"

const storage = new sst.cloudflare.Bucket("EnterpriseStorage")

new sst.cloudflare.x.SolidStart("Teams", {
  domain: shortDomain,
  path: "packages/enterprise",
  buildCommand: "bun run build:cloudflare",
  environment: {
    TALON_STORAGE_ADAPTER: "r2",
    TALON_STORAGE_ACCOUNT_ID: sst.cloudflare.DEFAULT_ACCOUNT_ID,
    TALON_STORAGE_ACCESS_KEY_ID: SECRET.R2AccessKey.value,
    TALON_STORAGE_SECRET_ACCESS_KEY: SECRET.R2SecretKey.value,
    TALON_STORAGE_BUCKET: storage.name,
  },
})
