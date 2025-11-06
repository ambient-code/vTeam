/// <reference types="cypress" />

// Custom command to set auth token for all requests
declare global {
  namespace Cypress {
    interface Chainable {
      /**
       * Custom command to set Bearer token for API authentication
       * @example cy.setAuthToken('my-token-here')
       */
      setAuthToken(token: string): Chainable<void>
    }
  }
}

Cypress.Commands.add('setAuthToken', (token: string) => {
  // Intercept all HTTP requests (including fetch, XHR, etc) and add Authorization header
  cy.intercept('**', (req) => {
    req.headers['Authorization'] = `Bearer ${token}`
  }).as('authInterceptor')
})

// Add global beforeEach to re-apply auth token
beforeEach(() => {
  const token = Cypress.env('TEST_TOKEN')
  if (token) {
    // Intercept all requests in this test
    cy.intercept('**', (req) => {
      req.headers['Authorization'] = `Bearer ${token}`
    })
  }
})

// Prevent TypeScript from reading file as legacy script
export {}

