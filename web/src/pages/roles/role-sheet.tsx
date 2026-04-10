import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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
import type { Role } from "./types"

const schema = z.object({
  name: z.string().min(1, "角色名称不能为空").max(64),
  code: z.string().min(1, "角色编码不能为空").max(64),
  description: z.string().max(255).optional(),
})

type FormValues = z.infer<typeof schema>

interface RoleSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  role: Role | null
}

export function RoleSheet({ open, onOpenChange, role }: RoleSheetProps) {
  const queryClient = useQueryClient()
  const isEditing = role !== null

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: "", code: "", description: "" },
  })

  useEffect(() => {
    if (open) {
      if (role) {
        form.reset({
          name: role.name,
          code: role.code,
          description: role.description || "",
        })
      } else {
        form.reset({ name: "", code: "", description: "" })
      }
    }
  }, [open, role, form])

  const createMutation = useMutation({
    mutationFn: (values: FormValues) => api.post("/api/v1/roles", values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] })
      onOpenChange(false)
    },
  })

  const updateMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.put(`/api/v1/roles/${role!.id}`, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] })
      onOpenChange(false)
    },
  })

  function onSubmit(values: FormValues) {
    if (isEditing) {
      updateMutation.mutate(values)
    } else {
      createMutation.mutate(values)
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending
  const error = createMutation.error || updateMutation.error

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-md">
        <SheetHeader>
          <SheetTitle>{isEditing ? "编辑角色" : "新建角色"}</SheetTitle>
          <SheetDescription className="sr-only">
            {isEditing ? "修改角色信息" : "填写角色信息以创建新角色"}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-4 px-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>角色名称</FormLabel>
                  <FormControl>
                    <Input placeholder="如：管理员" {...field} />
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
                  <FormLabel>角色编码</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="如：admin"
                      disabled={isEditing && role?.isSystem}
                      {...field}
                    />
                  </FormControl>
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
                    <Input placeholder="角色描述（可选）" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {error && (
              <p className="text-sm text-destructive">{error.message}</p>
            )}

            <SheetFooter>
              <Button variant="outline" size="sm" type="button" onClick={() => onOpenChange(false)}>
                取消
              </Button>
              <Button type="submit" size="sm" disabled={isPending}>
                {isPending ? "保存中..." : "保存"}
              </Button>
            </SheetFooter>
          </form>
        </Form>
      </SheetContent>
    </Sheet>
  )
}
