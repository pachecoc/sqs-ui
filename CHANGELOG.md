## SQS UI 0.2.0

### Added
- Runtime “Change Queue” dialog (update queue name or URL without restart).

### Improved
- UI consistency & simplified state management (removed global state file).
- Advisory messaging after automatic refresh explaining SQS eventual consistency.
- Error panel styling and JSON display readability.

### Changed
- README reorganized with highlights, configuration tables, and API list.
- Dockerfile enriched with OCI image labels and explanatory comments.

### Notes
- SQS approximate metrics and visibility timeout behaviors may cause temporary divergences between queue info and fetched messages.
- Purge remains irreversible; use carefully.
