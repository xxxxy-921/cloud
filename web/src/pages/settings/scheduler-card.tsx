import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"

const schema = z.object({
  historyRetentionDays: z.number().int().min(0, "不能小于 0"),
  auditRetentionDaysAuth: z.number().int().min(0, "不能小于 0"),
  auditRetentionDaysOperation: z.number().int().min(0, "不能小于 0"),
})

type FormValues = z.infer<typeof schema>

interface SchedulerSettings extends FormValues {}

export function SchedulerCard() {
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ["settings", "scheduler"],
    queryFn: () => api.get<SchedulerSettings>("/api/v1/settings/scheduler"),
  })

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    values: data ?? {
      historyRetentionDays: 30,
      auditRetentionDaysAuth: 90,
      auditRetentionDaysOperation: 365,
    },
  })

  const mutation = useMutation({
    mutationFn: (values: FormValues) => api.put("/api/v1/settings/scheduler", values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings", "scheduler"] })
      form.reset(form.getValues())
    },
  })

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex h-32 items-center justify-center text-muted-foreground">
          加载中...
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>自动清理</CardTitle>
        <CardDescription>
          配置各类历史数据的保留天数，设为 0 表示永不清理
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit((v) => mutation.mutate(v))} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-3">
              <FormField
                control={form.control}
                name="historyRetentionDays"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>任务执行历史</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={0}
                        {...field}
                        onChange={(e) => field.onChange(e.target.valueAsNumber)}
                      />
                    </FormControl>
                    <FormDescription>定时任务执行记录</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="auditRetentionDaysAuth"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>登录活动日志</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={0}
                        {...field}
                        onChange={(e) => field.onChange(e.target.valueAsNumber)}
                      />
                    </FormControl>
                    <FormDescription>登录、登出等认证记录</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="auditRetentionDaysOperation"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>操作记录日志</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={0}
                        {...field}
                        onChange={(e) => field.onChange(e.target.valueAsNumber)}
                      />
                    </FormControl>
                    <FormDescription>用户增删改等操作记录</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <Button
              type="submit"
              disabled={!form.formState.isDirty || mutation.isPending}
            >
              {mutation.isPending ? "保存中..." : "保存"}
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  )
}
