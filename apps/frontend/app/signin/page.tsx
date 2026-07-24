import { AuthForm } from "@/components/auth-form";
import { AuthLayout } from "@/components/auth-layout";

export default function SignInPage() {
  return (
    <AuthLayout
      eyebrow="Welcome back"
      title="Sign in to your studio"
      description="Continue analyzing videos and shaping your next standout clip."
    >
      <AuthForm mode="signin" />
    </AuthLayout>
  );
}
