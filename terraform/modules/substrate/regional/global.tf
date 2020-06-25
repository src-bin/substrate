module "global" {
  providers = { aws.global = aws.global }
  source    = "../global"
}
