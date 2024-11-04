anthropicClient := newAnthropicClient(os.Getenv("ANTHROPIC_API_KEY"))

summary := prompts.NewSummary(erSample)
prompt, _ := summary.Generate()

result, _ := anthropicClient.generate2(prompt)
log.Println(result)

apiResult := ApiResult{}
err = json.Unmarshal([]byte(result), &apiResult)
if err != nil {
  log.Fatal(err)
}
