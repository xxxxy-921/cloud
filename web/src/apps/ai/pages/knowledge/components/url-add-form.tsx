import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { useTranslation } from "react-i18next"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

function useUrlAddSchema() {
  const { t } = useTranslation("ai")
  return z.object({
    sourceUrl: z.string().min(1, t("validation.urlRequired")).url(t("validation.urlInvalid")),
    crawlDepth: z.coerce.number().int().min(0).max(2),
    urlPattern: z.string().max(256).optional(),
    crawlEnabled: z.boolean(),
    crawlSchedule: z.string().max(128).optional(),
  })
}

type FormValues = z.infer<ReturnType<typeof useUrlAddSchema>>

interface UrlAddFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  kbId: number
  onSuccess: () => void
}

export function UrlAddForm({ open, onOpenChange, kbId, onSuccess }: UrlAddFormProps) {
  const { t } = useTranslation(["ai", "common"])
  const schema = useUrlAddSchema()

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      sourceUrl: "",
      crawlDepth: 0,
      urlPattern: "",
      crawlEnabled: false,
      crawlSchedule: "",
    },
  })

  const watchCrawlEnabled = form.watch("crawlEnabled")

  useEffect(() => {
    if (open) {
      form.reset({
        sourceUrl: "",
        crawlDepth: 0,
        urlPattern: "",
        crawlEnabled: false,
        crawlSchedule: "",
      })
    }
  }, [open, form])

  const addMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.post(`/api/v1/ai/knowledge-bases/${kbId}/sources`, values),
    onSuccess: () => {
      onSuccess()
      onOpenChange(false)
      toast.success(t("ai:knowledge.sources.addUrlSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  function onSubmit(values: FormValues) {
    addMutation.mutate(values)
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{t("ai:knowledge.sources.addUrl")}</SheetTitle>
          <SheetDescription className="sr-only">
            {t("ai:knowledge.sources.addUrl")}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-5 px-4">
            <FormField
              control={form.control}
              name="sourceUrl"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("ai:knowledge.sources.url")}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="https://example.com/docs"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="crawlDepth"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("ai:knowledge.sources.crawlDepth")}</FormLabel>
                  <Select
                    value={String(field.value)}
                    onValueChange={(v) => field.onChange(Number(v))}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="0">0 — {t("ai:knowledge.sources.crawlDepthDesc.0")}</SelectItem>
                      <SelectItem value="1">1 — {t("ai:knowledge.sources.crawlDepthDesc.1")}</SelectItem>
                      <SelectItem value="2">2 — {t("ai:knowledge.sources.crawlDepthDesc.2")}</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="urlPattern"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t("ai:knowledge.sources.urlPattern")}
                    <span className="ml-1 text-muted-foreground font-normal text-xs">
                      ({t("common:optional")})
                    </span>
                  </FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t("ai:knowledge.sources.urlPatternPlaceholder")}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="crawlEnabled"
              render={({ field }) => (
                <FormItem className="flex items-center justify-between rounded-lg border p-3">
                  <div className="space-y-0.5">
                    <FormLabel>{t("ai:knowledge.sources.crawlEnabled")}</FormLabel>
                  </div>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            {watchCrawlEnabled && (
              <FormField
                control={form.control}
                name="crawlSchedule"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("ai:knowledge.sources.crawlSchedule")}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t("ai:knowledge.sources.crawlSchedulePlaceholder")}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            <SheetFooter>
              <Button type="submit" size="sm" disabled={addMutation.isPending}>
                {addMutation.isPending ? t("common:saving") : t("common:create")}
              </Button>
            </SheetFooter>
          </form>
        </Form>
      </SheetContent>
    </Sheet>
  )
}
