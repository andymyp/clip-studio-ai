import { AuthForm } from "@/components/auth-form";
import { AuthLayout } from "@/components/auth-layout";

export default function SignUpPage() {
  return (
    <AuthLayout
      eyebrow="Start creating"
      title="Build your clip workflow"
      description="Create a secure workspace for finding and producing your strongest moments."
    >
      <AuthForm mode="signup" />
    </AuthLayout>
  );
}
