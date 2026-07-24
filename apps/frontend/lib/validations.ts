import { z } from "zod";

const credentialsSchema = z.object({
  email: z
    .string()
    .trim()
    .min(1, "Email is required.")
    .email("Enter a valid email address.")
    .max(320, "Email is too long."),
  password: z
    .string()
    .min(8, "Password must contain at least 8 characters.")
    .max(72, "Password must contain at most 72 characters."),
});

export const authSchema = credentialsSchema.extend({
  confirmPassword: z.string().optional(),
});

export function createAuthSchema(signingUp: boolean) {
  return authSchema.superRefine((values, context) => {
    if (!signingUp) return;
    if (!values.confirmPassword) {
      context.addIssue({
        code: "custom",
        path: ["confirmPassword"],
        message: "Confirm your password.",
      });
    } else if (values.password !== values.confirmPassword) {
      context.addIssue({
        code: "custom",
        path: ["confirmPassword"],
        message: "Passwords do not match.",
      });
    }
  });
}

export const videoSearchSchema = z.object({
  keyword: z
    .string()
    .trim()
    .min(2, "Search must contain at least 2 characters.")
    .max(100, "Search must contain at most 100 characters."),
});

export type AuthValues = z.infer<typeof authSchema>;
export type VideoSearchValues = z.infer<typeof videoSearchSchema>;
