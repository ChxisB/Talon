interface ImportMetaEnv {
  readonly TALON_CHANNEL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module "virtual:talon-server" {
  export namespace Server {
    export const listen: typeof import("../../../talon/dist/types/src/node").Server.listen
    export type Listener = import("../../../talon/dist/types/src/node").Server.Listener
  }
  export namespace Config {
    export const get: typeof import("../../../talon/dist/types/src/node").Config.get
    export type Info = import("../../../talon/dist/types/src/node").Config.Info
  }
  export const bootstrap: typeof import("../../../talon/dist/types/src/node").bootstrap
}
