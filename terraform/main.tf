terraform { 
  cloud { 
    
    organization = "demo-tfe-org" 

    workspaces { 
      name = "workspace-test-1" 
    } 
  } 
}

