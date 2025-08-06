#!/bin/bash
# EXACT COPY of main logic from original workflow

# Source required functions
source "$(dirname "$0")/process-repos.sh"
source "$(dirname "$0")/send-webhook.sh"

# Final summary (EXACT COPY from original workflow)
echo ""
echo "📊 Final Summary:"
echo "  Total repositories: $TOTAL_REPOS"
echo "  Successfully backed up: $SUCCESS_COUNT"
echo "  Failed: $FAIL_COUNT"

# Send webhook notification (EXACT COPY from original workflow)
if [ $FAIL_COUNT -eq 0 ]; then
  send_webhook true "Backup successful: All $SUCCESS_COUNT repositories backed up" "${SUCCESSFUL_REPOS%, }"
  echo ""
  echo "✅ Backup completed successfully!"
else
  send_webhook false "Backup completed with errors: $SUCCESS_COUNT succeeded, $FAIL_COUNT failed (${FAILED_REPOS%, })" "${SUCCESSFUL_REPOS%, }"
  echo ""
  echo "⚠️ Backup completed with $FAIL_COUNT failures"
  exit 1
fi 