import { runtimeModules as threeRuntimeModules } from "@tui/three/runtime-modules"
import { ensureRuntimePluginSupport } from "@tui/solid/runtime-plugin-support/configure"

ensureRuntimePluginSupport({
  additional: threeRuntimeModules,
})
