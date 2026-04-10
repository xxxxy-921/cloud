export interface ConfigField {
  key: string
  label: string
  type: "string" | "number" | "boolean"
  required?: boolean
  default?: unknown
  sensitive?: boolean
  placeholder?: string
}

export interface ChannelTypeDef {
  label: string
  icon: string
  configSchema: ConfigField[]
}

export const CHANNEL_TYPES: Record<string, ChannelTypeDef> = {
  email: {
    label: "邮件 (SMTP)",
    icon: "Mail",
    configSchema: [
      { key: "host", label: "SMTP 服务器", type: "string", required: true, placeholder: "smtp.example.com" },
      { key: "port", label: "端口", type: "number", required: true, default: 465 },
      { key: "secure", label: "SSL/TLS", type: "boolean", default: true },
      { key: "username", label: "用户名", type: "string", required: true, placeholder: "user@example.com" },
      { key: "password", label: "密码", type: "string", required: true, sensitive: true },
      { key: "from", label: "发件人", type: "string", required: true, placeholder: "系统通知 <noreply@example.com>" },
    ],
  },
}
