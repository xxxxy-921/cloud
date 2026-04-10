import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"

const schema = z.object({
  to: z.string().email("请输入有效的邮箱地址"),
  subject: z.string().min(1, "主题不能为空"),
  body: z.string().min(1, "内容不能为空"),
})

type FormValues = z.infer<typeof schema>

interface SendTestDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  channelId: number | null
}

export function SendTestDialog({ open, onOpenChange, channelId }: SendTestDialogProps) {
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { to: "", subject: "Metis 测试邮件", body: "这是一封来自 Metis 系统的测试邮件。" },
  })

  const sendMutation = useMutation({
    mutationFn: async (values: FormValues) => {
      const res = await api.post<{ success: boolean; error?: string }>(
        `/api/v1/channels/${channelId}/send-test`,
        values,
      )
      if (!res.success) throw new Error(res.error || "发送失败")
      return res
    },
    onSuccess: () => {
      toast.success("测试邮件发送成功")
      onOpenChange(false)
    },
    onError: (err) => toast.error(`发送失败: ${err.message}`),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>发送测试邮件</DialogTitle>
          <DialogDescription>
            发送一封测试邮件以验证通道配置是否正常工作
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit((v) => sendMutation.mutate(v))} className="space-y-4">
            <FormField
              control={form.control}
              name="to"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>收件人</FormLabel>
                  <FormControl>
                    <Input placeholder="test@example.com" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="subject"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>主题</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="body"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>内容</FormLabel>
                  <FormControl>
                    <Textarea rows={3} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button type="submit" size="sm" disabled={sendMutation.isPending}>
                {sendMutation.isPending && <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />}
                发送
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
