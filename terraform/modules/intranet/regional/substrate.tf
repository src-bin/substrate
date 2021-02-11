module "substrate" {
  providers = {
    aws         = aws
    aws.network = aws.network
  }
  source = "../../substrate/regional"
}
