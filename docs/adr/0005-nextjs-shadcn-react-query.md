# ADR-0005: Next.js with Shadcn UI and React Query

**Status:** Accepted
**Date:** 2024-11-21
**Deciders:** Frontend Team
**Technical Story:** Frontend technology stack selection

## Context and Problem Statement

We need to build a modern web UI for the Ambient Code Platform with:

- Server-side rendering for fast initial loads
- Rich interactive components (session monitoring, project management)
- Real-time updates for session status
- Type-safe API integration
- Responsive design with accessible components

What frontend framework and UI library should we use?

## Decision Drivers

- **Modern patterns:** Server components, streaming, type safety
- **Developer experience:** Good tooling, active community
- **UI quality:** Professional design system, accessibility
- **Performance:** Fast initial load, efficient updates
- **Data fetching:** Caching, optimistic updates, real-time sync
- **Team expertise:** React knowledge on team

## Considered Options

1. **Next.js 14 + Shadcn UI + React Query (chosen)**
2. **Create React App + Material-UI + Redux**
3. **Remix + Chakra UI + React Query**
4. **Svelte/SvelteKit + Custom components**

## Decision Outcome

Chosen option: "Next.js 14 + Shadcn UI + React Query", because:

**Next.js 14 (App Router):**

1. **Server components:** Reduced client bundle size
2. **Streaming:** Progressive page rendering
3. **File-based routing:** Intuitive project structure
4. **TypeScript:** First-class type safety
5. **Industry momentum:** Large ecosystem, active development

**Shadcn UI:**

1. **Copy-paste components:** Own your component code
2. **Built on Radix UI:** Accessibility built-in
3. **Tailwind CSS:** Utility-first styling
4. **Customizable:** Full control over styling
5. **No runtime dependency:** Just copy components you need

**React Query:**

1. **Declarative data fetching:** Clean component code
2. **Automatic caching:** Reduces API calls
3. **Optimistic updates:** Better UX
4. **Real-time sync:** Easy integration with WebSockets
5. **DevTools:** Excellent debugging experience

### Consequences

**Positive:**

- **Performance:**
  - Server components reduce client JS by ~40%
  - React Query caching reduces redundant API calls
  - Streaming improves perceived performance

- **Developer Experience:**
  - TypeScript end-to-end (API to UI)
  - Shadcn components copy-pasted and owned
  - React Query hooks simplify data management
  - Next.js DevTools for debugging

- **User Experience:**
  - Fast initial page loads (SSR)
  - Smooth client-side navigation
  - Accessible components (WCAG 2.1 AA)
  - Responsive design (mobile-first)

**Negative:**

- **Learning curve:**
  - Next.js App Router is new (released 2023)
  - Server vs. client component mental model
  - React Query concepts (queries, mutations, invalidation)

- **Complexity:**
  - More moving parts than simple SPA
  - Server component restrictions (no hooks, browser APIs)
  - Hydration errors if server/client mismatch

**Risks:**

- Next.js App Router still evolving (breaking changes possible)
- Shadcn UI components need manual updates (not npm package)
- React Query cache invalidation can be tricky

## Implementation Notes

**Technology Versions:**

- Next.js: 14.x (App Router)
- React: 18.x
- Shadcn UI: Latest (no version, copy-paste)
- TanStack React Query: 5.x
- Tailwind CSS: 3.x
- TypeScript: 5.x

**Key Files:**
- `components/frontend/DESIGN_GUIDELINES.md` - Comprehensive patterns
- `components/frontend/src/components/ui/` - Shadcn components
- `components/frontend/src/services/queries/` - React Query hooks
- `components/frontend/src/app/` - Next.js pages

## Validation

**Performance Metrics:**

- Initial page load: <2s (Lighthouse score >90)
- Client bundle size: <200KB (with code splitting)
- Time to Interactive: <3s
- API call reduction: 60% fewer calls (React Query caching)

**Developer Feedback:**

- Positive: React Query simplifies data management significantly
- Positive: Shadcn components easy to customize
- Challenge: Server component restrictions initially confusing
- Resolution: Clear guidelines in DESIGN_GUIDELINES.md

**User Feedback:**

- Fast perceived performance (streaming)
- Smooth interactions (optimistic updates)
- Accessible (keyboard navigation, screen readers)

## Links

- Related: ADR-0004 (Go Backend with Python Runner)
- [Next.js 14 Documentation](https://nextjs.org/docs)
- [Shadcn UI](https://ui.shadcn.com/)
- [TanStack React Query](https://tanstack.com/query/latest)
- Frontend Guidelines: `components/frontend/DESIGN_GUIDELINES.md`
