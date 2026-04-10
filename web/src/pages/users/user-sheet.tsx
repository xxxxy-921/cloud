import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { api, type PaginatedResponse } from "@/lib/api"
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import type { User } from "@/stores/auth"

interface RoleOption {
  id: number
  name: string
  code: string
}

const createSchema = z.object({
  username: z.string().min(1, "用户名不能为空").max(64),
  password: z.string().min(1, "密码不能为空"),
  email: z.string().email("邮箱格式不正确").or(z.literal("")).optional(),
  phone: z.string().max(32).optional(),
  roleId: z.coerce.number().min(1, "请选择角色"),
})

const editSchema = z.object({
  email: z.string().email("邮箱格式不正确").or(z.literal("")).optional(),
  phone: z.string().max(32).optional(),
  roleId: z.coerce.number().min(1, "请选择角色"),
})

type CreateValues = z.infer<typeof createSchema>
type EditValues = z.infer<typeof editSchema>

interface UserSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: User | null
}

export function UserSheet({ open, onOpenChange, user }: UserSheetProps) {
  const queryClient = useQueryClient()
  const isEditing = user !== null

  const { data: rolesData } = useQuery({
    queryKey: ["roles", "all"],
    queryFn: () =>
      api.get<PaginatedResponse<RoleOption>>("/api/v1/roles?page=1&pageSize=100"),
    enabled: open,
  })

  const roles = rolesData?.items ?? []

  const form = useForm<CreateValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver((isEditing ? editSchema : createSchema) as any),
    defaultValues: {
      username: "",
      password: "",
      email: "",
      phone: "",
      roleId: 0,
    },
  })

  useEffect(() => {
    if (open) {
      if (user) {
        form.reset({
          username: user.username,
          password: "",
          email: user.email || "",
          phone: user.phone || "",
          roleId: user.role?.id || 0,
        })
      } else {
        form.reset({
          username: "",
          password: "",
          email: "",
          phone: "",
          roleId: 0,
        })
      }
    }
  }, [open, user, form])

  const createMutation = useMutation({
    mutationFn: (values: CreateValues) =>
      api.post("/api/v1/users", values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] })
      onOpenChange(false)
    },
  })

  const updateMutation = useMutation({
    mutationFn: (values: EditValues) =>
      api.put(`/api/v1/users/${user!.id}`, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] })
      onOpenChange(false)
    },
  })

  function onSubmit(values: CreateValues) {
    if (isEditing) {
      updateMutation.mutate({
        email: values.email,
        phone: values.phone,
        roleId: values.roleId,
      })
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
          <SheetTitle>{isEditing ? "编辑用户" : "新建用户"}</SheetTitle>
          <SheetDescription className="sr-only">
            {isEditing ? "修改用户信息" : "填写用户信息以创建新用户"}
          </SheetDescription>
        </SheetHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-4 px-4">
            <FormField
              control={form.control}
              name="username"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>用户名</FormLabel>
                  <FormControl>
                    <Input placeholder="用户名" disabled={isEditing} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {!isEditing && (
              <FormField
                control={form.control}
                name="password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>密码</FormLabel>
                    <FormControl>
                      <Input type="password" placeholder="密码" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>邮箱</FormLabel>
                  <FormControl>
                    <Input placeholder="邮箱（可选）" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="phone"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>手机号</FormLabel>
                  <FormControl>
                    <Input placeholder="手机号（可选）" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="roleId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>角色</FormLabel>
                  <Select
                    value={field.value ? String(field.value) : ""}
                    onValueChange={(v) => field.onChange(Number(v))}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="请选择角色" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {roles.map((role) => (
                        <SelectItem key={role.id} value={String(role.id)}>
                          {role.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
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
