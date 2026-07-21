export type SSEEventHandlers = Record<string, (payload: unknown) => void>

/**
 * Opens a Server-Sent Events connection and dispatches each named event to
 * its handler. EventSource already retries on connection loss natively, so
 * no manual reconnect logic is needed here.
 */
export function connectToStream(url: string, handlers: SSEEventHandlers): () => void {
    const eventSource = new EventSource(url)

    const listeners = Object.entries(handlers).map(([eventName, handler]) => {
        const listener = (event: MessageEvent) => dispatch(eventName, event, handler)
        eventSource.addEventListener(eventName, listener)
        return { eventName, listener }
    })

    return () => {
        for (const { eventName, listener } of listeners) {
            eventSource.removeEventListener(eventName, listener)
        }
        eventSource.close()
    }
}

function dispatch(eventName: string, event: MessageEvent, handler: (payload: unknown) => void): void {
    try {
        handler(JSON.parse(event.data))
    } catch (error) {
        console.error(`failed to parse SSE event "${eventName}"`, error)
    }
}