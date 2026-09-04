package dev.lumen.spike

import java.nio.file.Files
import java.nio.file.Path
import kotlin.time.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs

private val fixtures = Path.of("../fixtures")
private val evaluationTime = Instant.parse("2026-09-04T12:00:00Z")

private fun fixture(name: String) = Files.readString(fixtures.resolve(name))

class ProtocolTest {
    @Test fun validRequestAccepted() {
        assertIs<PairingResult.Accepted>(parsePairingRequest(fixture("pairing-request.valid.json"), evaluationTime))
    }

    @Test fun expiredRequestRejected() {
        assertIs<PairingResult.Rejected>(parsePairingRequest(fixture("pairing-request.expired.json"), evaluationTime))
    }

    @Test fun unsupportedVersionRejected() {
        assertIs<PairingResult.Rejected>(parsePairingRequest(fixture("pairing-request.unsupported-version.json"), evaluationTime))
    }

    @Test fun unknownTopLevelFieldRejected() {
        assertIs<PairingResult.Rejected>(parsePairingRequest(fixture("pairing-request.unknown-field.json"), evaluationTime))
    }

    @Test fun issuedAtAfterExpiresAtRejected() {
        val invalidOrdering = fixture("pairing-request.valid.json")
            .replace("2026-09-04T11:59:00Z", "2026-09-04T12:06:00Z")
            .replace("2026-09-04T12:05:00Z", "2026-09-04T12:05:00Z")
        assertIs<PairingResult.Rejected>(parsePairingRequest(invalidOrdering, evaluationTime))
    }

    @Test fun unknownNestedPayloadFieldRejected() {
        val unknownPayloadField = fixture("pairing-request.valid.json")
            .replace("\"displayName\": \"Synthetic Mac node\",", "\"displayName\": \"Synthetic Mac node\",\n    \"untrustedHint\": true,")
        assertIs<PairingResult.Rejected>(parsePairingRequest(unknownPayloadField, evaluationTime))
    }

    @Test fun ssePreservesEnvelopeFields() {
        assertEquals(
            listOf(
                TaskEvent("event_demo0001", "task.status", "{\"taskId\":\"task_demo0001\",\"state\":\"running\"}"),
                TaskEvent("event_demo0002", "task.status", "{\"taskId\":\"task_demo0001\",\"state\":\"awaiting_permission\"}"),
                TaskEvent("event_demo0003", "task.status", "{\"taskId\":\"task_demo0001\",\n\"state\":\"completed\"}")
            ),
            parseTaskEventsSse(fixture("task-events.sse"))
        )
    }
}
