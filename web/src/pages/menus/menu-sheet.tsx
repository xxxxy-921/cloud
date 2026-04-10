import { useEffect, useMemo } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
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
import type { MenuItem } from "@/stores/menu"

const schema = z.object({
  parentId: z.coerce.number().nullable(),
  name: z.string().min(1, "菜单名称不能为空").max(64),
  type: z.enum(["directory", "menu", "button"]),
  path: z.string().max(255).optional(),
  icon: z.string().max(64).optional(),
  permission: z.string().max(128).optional(),
  sort: z.coerce.number().int().min(0).default(0),
  isHidden: z.boolean().default(false),
})

type FormValues = z.infer<typeof schema>

interface MenuSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  menu: MenuItem | null
  parentId?: number | null
}

function flattenForSelect(menus: MenuItem[], depth = 0): { id: number; label: string }[] {
  const result: { id: number; label: string }[] = []
  for (const m of menus) {
    if (m.type !== "button") {
      result.push({ id: m.id, label: "\u00A0\u00A0".repeat(depth) + m.name })
      if (m.children) result.push(...flattenForSelect(m.children, depth + 1))
    }
  }
  return result
}

export function MenuSheet({ open, onOpenChange, menu, parentId }: MenuSheetProps) {
  const queryClient = useQueryClient()
  const isEditing = menu !== null

  const { data: menuTree } = useQuery({
    queryKey: ["menus", "tree"],
    queryFn: () => api.get<MenuItem[]>("/api/v1/menus/tree"),
    enabled: open,
  })

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      parentId: null,
      name: "",
      type: "menu" as const,
      path: "",
      icon: "",
      permission: "",
      sort: 0,
      isHidden: false,
    },
  })

  useEffect(() => {
    if (open) {
      if (menu) {
        form.reset({
          parentId: menu.parentId,
          name: menu.name,
          type: menu.type,
          path: menu.path || "",
          icon: menu.icon || "",
          permission: menu.permission || "",
          sort: menu.sort,
          isHidden: menu.isHidden,
        })
      } else {
        form.reset({
          parentId: parentId ?? null,
          name: "",
          type: "menu",
          path: "",
          icon: "",
          permission: "",
          sort: 0,
          isHidden: false,
        })
      }
    }
  }, [open, menu, parentId, form])

  const parentOptions = useMemo(
    () => (menuTree ? flattenForSelect(menuTree) : []),
    [menuTree],
  )

  const createMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.post("/api/v1/menus", {
        ...values,
        parentId: values.parentId ?? undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["menus"] })
      onOpenChange(false)
    },
  })

  const updateMutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.put(`/api/v1/menus/${menu!.id}`, {
        ...values,
        parentId: values.parentId ?? undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["menus"] })
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
  const menuType = form.watch("type")

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-md overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{isEditing ? "编辑菜单" : "新建菜单"}</SheetTitle>
          <SheetDescription className="sr-only">
            {isEditing ? "修改菜单项信息" : "填写菜单信息以创建新菜单项"}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-4 px-4">
            <FormField
              control={form.control}
              name="parentId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>上级菜单</FormLabel>
                  <Select
                    value={field.value != null ? String(field.value) : ""}
                    onValueChange={(v) => field.onChange(v ? Number(v) : null)}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="顶级菜单" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="">顶级菜单</SelectItem>
                      {parentOptions.map((opt) => (
                        <SelectItem key={opt.id} value={String(opt.id)}>{opt.label}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="type"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>类型</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="directory">目录</SelectItem>
                      <SelectItem value="menu">菜单</SelectItem>
                      <SelectItem value="button">按钮</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>名称</FormLabel>
                  <FormControl>
                    <Input placeholder="菜单名称" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {menuType !== "button" && (
              <>
                <FormField
                  control={form.control}
                  name="path"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>路径</FormLabel>
                      <FormControl>
                        <Input placeholder="路由路径，如 /users" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="icon"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>图标</FormLabel>
                      <FormControl>
                        <Input placeholder="图标名称（可选）" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </>
            )}
            <FormField
              control={form.control}
              name="permission"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>权限标识</FormLabel>
                  <FormControl>
                    <Input placeholder="如 system:user:list" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="isHidden"
              render={({ field }) => (
                <FormItem className="flex items-center gap-2">
                  <FormControl>
                    <Checkbox
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <FormLabel className="!mt-0">隐藏菜单</FormLabel>
                </FormItem>
              )}
            />

            {error && (
              <p className="text-sm text-destructive">{error.message}</p>
            )}

            <SheetFooter>
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
