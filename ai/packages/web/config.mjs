const stage = process.env.SST_STAGE || "dev"

export default {
  url: stage === "production" ? "https://talon.ai" : `https://${stage}.talon.ai`,
  console: stage === "production" ? "https://talon.ai/auth" : `https://${stage}.talon.ai/auth`,
  email: "help@anoma.ly",
  socialCard: "https://social-cards.sst.dev",
  github: "https://github.com/ChxisB/Talon",
  discord: "https://talon.ai/discord",
  headerLinks: [
    { name: "app.header.home", url: "/" },
    { name: "app.header.docs", url: "/docs/" },
  ],
}
