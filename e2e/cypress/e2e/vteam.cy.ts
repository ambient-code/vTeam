describe('vTeam E2E Tests', () => {
  before(() => {
    // Verify auth token is available
    const token = Cypress.env('TEST_TOKEN')
    expect(token, 'TEST_TOKEN environment variable should be set').to.exist
    // Note: Auth header is automatically added via beforeEach in commands.ts
  })

  it('should access the UI with token authentication', () => {
    cy.visit('/')
    
    // Wait for the page to load and verify we see the Projects header
    cy.contains('Projects', { timeout: 15000 }).should('be.visible')
  })

  it('should navigate to new project page', () => {
    cy.visit('/projects')
    
    // Click the "New Project" button
    cy.contains('New Project').click()
    
    // Verify we're on the new project page
    cy.url().should('include', '/projects/new')
    cy.contains('Create New Project').should('be.visible')
  })

  it('should create a new project', () => {
    cy.visit('/projects/new')
    
    // Generate unique project name
    const projectName = `e2e-test-${Date.now()}`
    
    // Fill in project form
    cy.get('#name').clear().type(projectName)
    
    // Submit the form
    cy.contains('button', 'Create Project').click()
    
    // Verify redirect to project page
    cy.url({ timeout: 15000 }).should('include', `/projects/${projectName}`)
    cy.contains(projectName).should('be.visible')
  })

  it('should list the created projects', () => {
    cy.visit('/projects')
    
    // Wait for projects list to load
    cy.get('body', { timeout: 10000 }).should('be.visible')
    
    // Verify we can see projects (at least the one we created)
    cy.contains('Projects').should('be.visible')
  })

  it('should access backend API cluster-info endpoint', () => {
    // Test that backend API is accessible
    // Note: /health is at root level, not under /api
    // Auth header is added automatically via interceptor
    cy.request('/api/cluster-info').then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body).to.have.property('isOpenShift')
      expect(response.body.isOpenShift).to.eq(false)  // kind is vanilla k8s
    })
  })
})

