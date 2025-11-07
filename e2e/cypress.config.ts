import { defineConfig } from 'cypress'

export default defineConfig({
  e2e: {
    // Use CYPRESS_BASE_URL env var, fallback to default
    baseUrl: process.env.CYPRESS_BASE_URL || 'http://vteam.local',
    video: true,  // Enable video recording
    screenshotOnRunFailure: true,
    defaultCommandTimeout: 10000,
    requestTimeout: 10000,
    responseTimeout: 10000,
    viewportWidth: 1280,
    viewportHeight: 720,
    setupNodeEvents(on, config) {
      // implement node event listeners here if needed
    },
  },
})

