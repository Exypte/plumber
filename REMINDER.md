# Issue-361

1. Prise en main de plumber
      - Lecture du CONTRIBUTE.md
      - Clone du projet
      - Claude /init
      - Demande de résolution de l’issue 361
      - Lecture du plan proposé par Claude et du code plumber
2. Compréhension de l’issue 361
      - Identification des points important
        - componentMustComeFromAuthorizedSources
        - functionMustComeFromAuthorizedSources
        - Copier le fonctionnement de containerImageMustComeFromAuthorizedSources
      -	Lecture de la doc gitlab concernant les components et les functions
      -	Test sur l’API gitlab
3. Function and Components
   - Components and functions operate at different levels of the pipeline and solve different problems.
   - GitLab includes a component before any jobs run and contributes jobs, stages, and configuration to the pipeline.
   - GitLab Functions are reusable at the job level. They run inside a job and replace the script.
4. Functions
    - GitLab Functions was previously called CI/CD Steps. The feature and its syntax have been renamed.
    - Functions are loaded from the file system or an OCI repository. Loading from a Git repository is supported but deprecated.
      - OCI: registry.gitlab.com/gitlab-org/ci-cd/runner-tools/gitlab-functions-examples/echo:1
      - File Sys: ./path/to/my-function
      - Git repository (deprecated): gitlab.com/funcs/my-git-repo@v1.0.0
5. Components
   - Pipeline configuration and component configuration are not processed independently. When a pipeline starts, any included component configuration merges into the pipeline’s configuration. If your pipeline and the component both contain configuration with the same name, they can interact in unexpected ways.
6. Question que j'aurais dû poser
   - Faut-il que plumber est le même comportement en local que dans la CI ? 
   - Faut-il en plus de faire confiance au fournisseur de `GitLab functions` et de `GitLab components` vérifier leurs contenus ?
7. Proposition d'un plan

```text
Implémente l'issue #361 de GitHub.

L'issue concerne l'ajout de deux controls spécifiques à GitLab :
    1. Les GitLab CI/CD components : https://docs.gitlab.com/ci/components/
    2. Les GitLab CI/CD functions : https://docs.gitlab.com/ci/functions/

Globalement, l'implémentation ressemblera à containerImageMustComeFromAuthorizedSources.

L'output doit contenir le Total, les Authorized, les Unauthorized et les Deprecated.

Le nouveau control componentMustComeFromAuthorizedSources aura le code 414. Il sera activé par défaut et aura pour trustedUrls par défaut $CI_SERVER_FQDN/$CI_PROJECT_PATH/* et ${CI_SERVER_FQDN}/${CI_PROJECT_PATH}/*.

Le nouveau control functionMustComeFromAuthorizedSources aura le code 415. Il sera activé par défaut et aura pour trustedUrls par défaut $CI_TEMPLATE_REGISTRY_HOST/$CI_PROJECT_PATH/* et ${CI_TEMPLATE_REGISTRY_HOST}/${CI_PROJECT_PATH}/*.

L'include des components est résolu avant que plumber ne récupère le fichier .gitlab-ci.yml. Cela signifie que si des CI/CD variables sont présentes, leurs valeurs auront déjà été interprétées. Il faut donc que les variables présentes dans les trustedUrls soient interprétées avant le rego.

Le fonctionnement des functions reste standard, puisque c'est interprété seulement au moment du job.
```

---

```shell
 curl --request GET \                                                                                                                                                                    at  12:20:41 PM
  --header "PRIVATE-TOKEN: glpat-eiSkVUtCiOtkU_J4rBzP4mM6MQpvOjEKdTo1NXd0Mg8.01.171fmia9w" \
  --url "https://gitlab.com/api/v4/projects/Exypte%2Ftest-ci/variables" 
```

```shell
curl --request GET \                                                                                                                                                                    at  12:20:41 PM
  --header "PRIVATE-TOKEN: glpat-eiSkVUtCiOtkU_J4rBzP4mM6MQpvOjEKdTo1NXd0Mg8.01.171fmia9w" \
  --url "https://gitlab.com/api/v4/projects/Exypte%2Ftest-ci/repository/files/.gitlab-ci.yml?ref=main"
```