"use client"

import { useEffect, useMemo, useRef } from "react"
import { useChat, Chat, type UseChatHelpers } from "@ai-sdk/react"
import { useAISDKRuntime } from "@assistant-ui/react-ai-sdk"
import { DefaultChatTransport, type UIMessage } from "ai"

import { api, sessionApi, type SessionMessage } from "@/lib/api"
import { sessionMessagesToUIMessages } from "@/components/chat-workspace"

function sessionMessagesSignature(messages: SessionMessage[] | undefined) {
  if (!messages) return ""
  return messages
    .map((message) => {
      const metadata = message.metadata ? JSON.stringify(message.metadata) : ""
      return `${message.id}:${message.sequence}:${message.role}:${message.content}:${metadata}`
    })
    .join("|")
}

function fastSnapshot(message: UIMessage): UIMessage {
  return {
    ...message,
    parts: message.parts.map((part) => ({ ...part })),
  } as UIMessage
}

export interface UseServiceDeskChatOptions {
  onFinish?: () => void
  onError?: (error: Error) => void
}

export function useServiceDeskChat(
  sessionId: number,
  initialSessionMessages?: SessionMessage[],
  options?: UseServiceDeskChatOptions,
) {
  const optionsRef = useRef(options)
  useEffect(() => {
    optionsRef.current = options
  }, [options])

  const initialSignature = useMemo(
    () => sessionMessagesSignature(initialSessionMessages),
    [initialSessionMessages],
  )
  const initialMessages = useMemo(
    () => sessionMessagesToUIMessages(initialSessionMessages ?? []),
    [initialSessionMessages, initialSignature],
  )

  const transport = useMemo(() => {
    const authenticatedFetch: typeof fetch = (input, init) =>
      api.fetch(input instanceof Request ? input.url : String(input), init)

    return new DefaultChatTransport<UIMessage>({
      api: sessionApi.chatUrl(sessionId),
      fetch: authenticatedFetch,
    })
  }, [sessionId])

  const chatInstance = useMemo(() => {
    const chat = new Chat<UIMessage>({
      id: String(sessionId),
      messages: initialMessages,
      transport,
      onFinish: () => optionsRef.current?.onFinish?.(),
      onError: (error) => optionsRef.current?.onError?.(error),
    })
    ;(chat as unknown as { state: { snapshot: typeof fastSnapshot } }).state.snapshot = fastSnapshot
    return chat
  }, [initialMessages, sessionId, transport])

  const chat = useChat({
    chat: chatInstance,
  }) as UseChatHelpers<UIMessage>
  const runtime = useAISDKRuntime(chat)

  return { chat, runtime }
}
