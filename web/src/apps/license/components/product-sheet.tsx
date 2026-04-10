import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useNavigate } from "react-router"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
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

export interface ProductItem {
  id: number
  name: string
  code: string
  description: string
  status: string
  planCount: number
  createdAt: string
  updatedAt: string
}

const schema = z.object({
  name: z.string().min(1, "名称不能为空").max(128),
  code: z
    .string()
    .min(1, "编码不能为空")
    .max(64)
    .regex(/^[a-z0-9-]+$/, "仅允许小写字母、数字和连字符"),
  description: z.string().optional(),
})

type FormValues = z.infer<typeof schema>

interface ProductSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  product: ProductItem | null
}

export function ProductSheet({ open, onOpenChange, product }: ProductSheetProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const isEditing = product !== null

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: "", code: "", description: "" },
  })

  useEffect(() => {
    if (open) {
      if (product) {
        form.reset({ name: product.name, code: product.code, description: product.description })
      } else {
        form.reset({ name: "", code: "", description: "" })
      }
    }
  }, [open, product, form])

  const createMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.post<ProductItem>("/api/v1/license/products", values),
    onSuccess: (created) => {
      queryClient.invalidateQueries({ queryKey: ["license-products"] })
      onOpenChange(false)
      toast.success("商品创建成功")
      navigate(`/license/products/${created.id}?tab=schema`)
    },
    onError: (err) => toast.error(err.message),
  })

  const updateMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.put(`/api/v1/license/products/${product!.id}`, {
        name: values.name,
        description: values.description,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-products"] })
      queryClient.invalidateQueries({ queryKey: ["license-product"] })
      onOpenChange(false)
      toast.success("商品更新成功")
    },
    onError: (err) => toast.error(err.message),
  })

  function onSubmit(values: FormValues) {
    if (isEditing) {
      updateMutation.mutate(values)
    } else {
      createMutation.mutate(values)
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>{isEditing ? "编辑商品" : "新建商品"}</SheetTitle>
          <SheetDescription className="sr-only">
            {isEditing ? "编辑商品" : "新建商品"}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-5 px-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>商品名称</FormLabel>
                  <FormControl>
                    <Input placeholder="例如：NekoMonitor" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="code"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>商品编码</FormLabel>
                  {isEditing ? (
                    <div>
                      <Input value={field.value} disabled />
                      <p className="text-xs text-muted-foreground mt-1">编码创建后不可更改</p>
                    </div>
                  ) : (
                    <>
                      <FormControl>
                        <Input placeholder="例如：neko-monitor" {...field} />
                      </FormControl>
                      <p className="text-xs text-muted-foreground">
                        用于唯一标识商品，建议使用英文、小写和连字符。
                      </p>
                    </>
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>描述</FormLabel>
                  <FormControl>
                    <Textarea placeholder="商品描述（可选）" rows={3} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <SheetFooter>
              <Button type="submit" size="sm" disabled={isPending}>
                {isPending ? "保存中..." : isEditing ? "保存" : "创建"}
              </Button>
            </SheetFooter>
          </form>
        </Form>
      </SheetContent>
    </Sheet>
  )
}
