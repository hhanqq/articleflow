# Articleflow Web

Static SPA for the Articleflow feed.

Features:

- vertical article feed from `gateway-api`
- stored article search through `gateway-api`
- separate async parser job creation and polling for fresh source fetches
- reaction buttons for `save`, `like`, `skip`, `dislike`, and `open`
- editable API base URL stored in local storage

Run:

```bash
npm run start
```

Then open:

```text
http://127.0.0.1:5173
```

Tests:

```bash
npm test
```
