BUCKET_PREFIX := docs-substrate-tools
DOMAIN := www
QUALITY := gamma
REGION := us-east-1
ROLE := GitHubActions

all:
	mkdir -p www
	find -type d -printf %P\\n | grep -v ^.git | grep -v ^www | xargs -I_ mkdir -p www/_
	find -name \*.md -type f -printf %P\\n | grep -v ^.git | grep -v ^www | while read P; do mergician -o www/$${P%.md}.html $$P; done
	find -not -name \*.md -type f -printf %P\\n | grep -v ^.git | grep -v ^www | xargs -I_ cp _ www/_
	mv www/README.html www/index.html

clean:
	rm -f -r www
	find -name \*.html -delete
	find -name .\*.html.sha256 -delete

production:
	printf "User-agent: *\nAllow: /\n" | substrate assume-role --domain $(DOMAIN) --environment $@ --quality $(QUALITY) --role $(ROLE) aws s3 cp --region $(REGION) - s3://$(BUCKET_PREFIX)-$(DOMAIN)-$@-$(QUALITY)/robots.txt
	substrate assume-role --domain $(DOMAIN) --environment $@ --quality $(QUALITY) --role $(ROLE) aws s3 sync --delete --exclude Makefile --exclude robots.txt --exclude \*.md --exclude \*.sha256 --exclude \*.swp --region $(REGION) www s3://$(BUCKET_PREFIX)-$(DOMAIN)-$@-$(QUALITY)
	substrate assume-role --domain $(DOMAIN) --environment $@ --quality $(QUALITY) --role $(ROLE) aws cloudfront create-invalidation --distribution-id EGSA4A34M983F --paths /\*

staging:
	printf "User-agent: *\nDisallow: /\n" | substrate assume-role --domain $(DOMAIN) --environment $@ --quality $(QUALITY) --role $(ROLE) aws s3 cp --region $(REGION) - s3://$(BUCKET_PREFIX)-$(DOMAIN)-$@-$(QUALITY)/robots.txt
	substrate assume-role --domain $(DOMAIN) --environment $@ --quality $(QUALITY) --role $(ROLE) aws s3 sync --delete --exclude Makefile --exclude robots.txt --exclude \*.md --exclude \*.sha256 --exclude \*.swp --region $(REGION) www s3://$(BUCKET_PREFIX)-$(DOMAIN)-$@-$(QUALITY)

.PHONY: all clean production staging
