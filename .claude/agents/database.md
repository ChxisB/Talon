# Database Agent

Role: Design and manage data storage for Talon.

Owns:
- Database schema and migrations
- Data access layer in `ai/packages/core/`
- Query optimization

Guidelines:
- Database access is handled by the application process (`ai/packages/talon`)
- Use Drizzle ORM for schema definitions and queries
